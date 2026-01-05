package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepository struct {
	pool *pgxpool.Pool
}

func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{pool: pool}
}

func (r *AdminRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

func (r *AdminRepository) CreateEvent(ctx context.Context, event domain.Event) error {
	status := event.Status
	if status == "" {
		status = domain.EventStatusActive
	}
	const stmt = `
INSERT INTO events (id, name, starts_at, status, cancelled_at)
VALUES ($1, $2, $3, $4, $5)`
	_, err := r.exec(ctx, stmt, event.ID, event.Name, event.StartsAt, status, event.CancelledAt)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (r *AdminRepository) ListEvents(ctx context.Context) ([]domain.Event, error) {
	const query = `
SELECT id, name, starts_at, status, cancelled_at
FROM events
ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var event domain.Event
		var status string
		if err := rows.Scan(&event.ID, &event.Name, &event.StartsAt, &status, &event.CancelledAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.Status = domain.EventStatus(status)
		events = append(events, event)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate events: %w", rows.Err())
	}
	return events, nil
}

func (r *AdminRepository) CancelEvent(ctx context.Context, eventID string, now time.Time) (domain.Event, error) {
	var event domain.Event

	if err := withTx(ctx, r.pool, func(txCtx context.Context) error {
		const selectStmt = `
SELECT id, name, starts_at, status, cancelled_at
FROM events
WHERE id = $1
FOR UPDATE`

		var status string
		row := txFromContext(txCtx).QueryRow(txCtx, selectStmt, eventID)
		if err := row.Scan(&event.ID, &event.Name, &event.StartsAt, &status, &event.CancelledAt); err != nil {
			if isInvalidUUID(err) {
				return domain.ErrInvalidID
			}
			if err == pgx.ErrNoRows {
				return domain.ErrEventNotFound
			}
			return fmt.Errorf("select event: %w", err)
		}
		event.Status = domain.EventStatus(status)

		if event.Status != domain.EventStatusCancelled {
			const updateEvent = `
UPDATE events
SET status = 'cancelled', cancelled_at = $2, updated_at = $2
WHERE id = $1`
			if _, err := txFromContext(txCtx).Exec(txCtx, updateEvent, eventID, now); err != nil {
				return fmt.Errorf("update event: %w", err)
			}

			const updateHolds = `
UPDATE holds
SET status = 'invalid'
WHERE event_id = $1 AND status IN ('active', 'confirmed')`
			if _, err := txFromContext(txCtx).Exec(txCtx, updateHolds, eventID); err != nil {
				return fmt.Errorf("invalidate holds: %w", err)
			}

			const updateOrders = `
UPDATE orders
SET status = 'refunded'
WHERE hold_id IN (SELECT id FROM holds WHERE event_id = $1) AND status = 'confirmed'`
			if _, err := txFromContext(txCtx).Exec(txCtx, updateOrders, eventID); err != nil {
				return fmt.Errorf("refund orders: %w", err)
			}

			event.Status = domain.EventStatusCancelled
			event.CancelledAt = &now
		}

		return nil
	}); err != nil {
		return domain.Event{}, err
	}

	return event, nil
}

func (r *AdminRepository) GetEventForUpdate(ctx context.Context, eventID string) (domain.Event, error) {
	const query = `
SELECT id, name, starts_at, status, cancelled_at
FROM events
WHERE id = $1
FOR UPDATE`

	var event domain.Event
	var status string
	if err := r.queryRow(ctx, query, eventID).Scan(&event.ID, &event.Name, &event.StartsAt, &status, &event.CancelledAt); err != nil {
		if isInvalidUUID(err) {
			return domain.Event{}, domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return domain.Event{}, domain.ErrEventNotFound
		}
		return domain.Event{}, fmt.Errorf("get event: %w", err)
	}
	event.Status = domain.EventStatus(status)
	return event, nil
}

func (r *AdminRepository) UpdateEventStatus(ctx context.Context, eventID string, status domain.EventStatus) error {
	const stmt = `UPDATE events SET status = $2 WHERE id = $1`

	tag, err := r.exec(ctx, stmt, eventID, status)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("update event status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEventNotFound
	}
	return nil
}

func (r *AdminRepository) CreateZone(ctx context.Context, zone domain.Zone) error {
	const stmt = `
INSERT INTO zones (id, event_id, name, capacity)
VALUES ($1, $2, $3, $4)`
	_, err := r.exec(ctx, stmt, zone.ID, zone.EventID, zone.Name, zone.Capacity)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		if isUniqueViolation(err) {
			return domain.ErrZoneAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return domain.ErrEventNotFound
		}
		return fmt.Errorf("create zone: %w", err)
	}
	return nil
}

func (r *AdminRepository) ListZonesByEvent(ctx context.Context, eventID string, now time.Time) ([]domain.Zone, error) {
	const existsQuery = `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`
	var exists bool
	if err := r.pool.QueryRow(ctx, existsQuery, eventID).Scan(&exists); err != nil {
		if isInvalidUUID(err) {
			return nil, domain.ErrInvalidID
		}
		return nil, fmt.Errorf("check event: %w", err)
	}
	if !exists {
		return nil, domain.ErrEventNotFound
	}

	const query = `
SELECT z.id, z.event_id, z.name, z.capacity,
       z.capacity
         - COALESCE(SUM(CASE WHEN h.status = 'active' AND h.expires_at > $2 THEN h.quantity ELSE 0 END), 0)
         - COALESCE(SUM(CASE WHEN h.status = 'confirmed' THEN h.quantity ELSE 0 END), 0) AS available
FROM zones z
LEFT JOIN holds h ON h.zone_id = z.id
WHERE z.event_id = $1
GROUP BY z.id, z.event_id, z.name, z.capacity, z.created_at
ORDER BY z.created_at ASC`
	rows, err := r.pool.Query(ctx, query, eventID, now)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	defer rows.Close()

	var zones []domain.Zone
	for rows.Next() {
		var zone domain.Zone
		if err := rows.Scan(&zone.ID, &zone.EventID, &zone.Name, &zone.Capacity, &zone.Available); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		zones = append(zones, zone)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate zones: %w", rows.Err())
	}
	return zones, nil
}

func (r *AdminRepository) ListActiveHoldsByZone(ctx context.Context, eventID, zoneID string, now time.Time) ([]domain.Hold, error) {
	if err := r.ensureEventAndZone(ctx, eventID, zoneID); err != nil {
		return nil, err
	}

	const query = `
SELECT id, event_id, zone_id, quantity, status, expires_at, idempotency_key, created_at
FROM holds
WHERE event_id = $1 AND zone_id = $2 AND status = 'active' AND expires_at > $3
ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query, eventID, zoneID, now)
	if err != nil {
		return nil, fmt.Errorf("list active holds: %w", err)
	}
	defer rows.Close()

	holds := []domain.Hold{}
	for rows.Next() {
		var hold domain.Hold
		var status string
		if err := rows.Scan(
			&hold.ID,
			&hold.EventID,
			&hold.ZoneID,
			&hold.Quantity,
			&status,
			&hold.ExpiresAt,
			&hold.IdempotencyKey,
			&hold.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hold: %w", err)
		}
		hold.Status = domain.HoldStatus(status)
		holds = append(holds, hold)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate holds: %w", rows.Err())
	}
	return holds, nil
}

func (r *AdminRepository) ListOrdersByZone(ctx context.Context, eventID, zoneID string) ([]domain.Order, error) {
	if err := r.ensureEventAndZone(ctx, eventID, zoneID); err != nil {
		return nil, err
	}

	const query = `
SELECT o.id, o.hold_id, o.idempotency_key, o.status, o.created_at
FROM orders o
JOIN holds h ON h.id = o.hold_id
WHERE h.event_id = $1 AND h.zone_id = $2 AND o.status = 'confirmed'
ORDER BY o.created_at ASC`
	rows, err := r.pool.Query(ctx, query, eventID, zoneID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := []domain.Order{}
	for rows.Next() {
		var order domain.Order
		var status string
		if err := rows.Scan(&order.ID, &order.HoldID, &order.IdempotencyKey, &status, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		order.Status = domain.OrderStatus(status)
		orders = append(orders, order)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate orders: %w", rows.Err())
	}
	return orders, nil
}

func (r *AdminRepository) ensureEventAndZone(ctx context.Context, eventID, zoneID string) error {
	const eventQuery = `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`
	var eventExists bool
	if err := r.pool.QueryRow(ctx, eventQuery, eventID).Scan(&eventExists); err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("check event: %w", err)
	}
	if !eventExists {
		return domain.ErrEventNotFound
	}

	const zoneQuery = `SELECT EXISTS (SELECT 1 FROM zones WHERE id = $1 AND event_id = $2)`
	var zoneExists bool
	if err := r.pool.QueryRow(ctx, zoneQuery, zoneID, eventID).Scan(&zoneExists); err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("check zone: %w", err)
	}
	if !zoneExists {
		return domain.ErrZoneNotFound
	}
	return nil
}

func (r *AdminRepository) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return r.pool.Exec(ctx, sql, args...)
}

func (r *AdminRepository) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return r.pool.QueryRow(ctx, sql, args...)
}
