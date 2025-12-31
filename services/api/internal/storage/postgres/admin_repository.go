package postgres

import (
	"context"
	"fmt"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepository struct {
	pool *pgxpool.Pool
}

func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{pool: pool}
}

func (r *AdminRepository) CreateEvent(ctx context.Context, event domain.Event) error {
	const stmt = `
INSERT INTO events (id, name, starts_at)
VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, stmt, event.ID, event.Name, event.StartsAt)
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
SELECT id, name, starts_at
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
		if err := rows.Scan(&event.ID, &event.Name, &event.StartsAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, event)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate events: %w", rows.Err())
	}
	return events, nil
}

func (r *AdminRepository) CreateZone(ctx context.Context, zone domain.Zone) error {
	const stmt = `
INSERT INTO zones (id, event_id, name, capacity)
VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, stmt, zone.ID, zone.EventID, zone.Name, zone.Capacity)
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

func (r *AdminRepository) ListZonesByEvent(ctx context.Context, eventID string) ([]domain.Zone, error) {
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
SELECT id, event_id, name, capacity
FROM zones
WHERE event_id = $1
ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	defer rows.Close()

	var zones []domain.Zone
	for rows.Next() {
		var zone domain.Zone
		if err := rows.Scan(&zone.ID, &zone.EventID, &zone.Name, &zone.Capacity); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		zones = append(zones, zone)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate zones: %w", rows.Err())
	}
	return zones, nil
}
