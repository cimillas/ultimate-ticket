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

type HoldRepository struct {
	pool *pgxpool.Pool
}

func NewHoldRepository(pool *pgxpool.Pool) *HoldRepository {
	return &HoldRepository{pool: pool}
}

func (r *HoldRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

func (r *HoldRepository) GetZoneForUpdate(ctx context.Context, eventID, zoneID string) (domain.Zone, error) {
	const query = `SELECT id, event_id, name, capacity FROM zones WHERE id = $1 AND event_id = $2 FOR UPDATE`
	var z domain.Zone
	err := r.queryRow(ctx, query, zoneID, eventID).Scan(&z.ID, &z.EventID, &z.Name, &z.Capacity)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.Zone{}, domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return domain.Zone{}, domain.ErrZoneNotFound
		}
		return domain.Zone{}, fmt.Errorf("get zone: %w", err)
	}
	return z, nil
}

func (r *HoldRepository) FindHoldByIdempotencyKey(ctx context.Context, eventID, zoneID, key string) (*domain.Hold, error) {
	const query = `
SELECT id, event_id, zone_id, quantity, status, expires_at, idempotency_key, created_at
FROM holds
WHERE event_id = $1 AND zone_id = $2 AND idempotency_key = $3`

	var h domain.Hold
	err := r.queryRow(ctx, query, eventID, zoneID, key).
		Scan(&h.ID, &h.EventID, &h.ZoneID, &h.Quantity, &h.Status, &h.ExpiresAt, &h.IdempotencyKey, &h.CreatedAt)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find hold by idempotency key: %w", err)
	}
	return &h, nil
}

func (r *HoldRepository) SumActiveHolds(ctx context.Context, eventID, zoneID string, now time.Time) (int, error) {
	const query = `
SELECT COALESCE(SUM(quantity), 0)
FROM holds
WHERE event_id = $1 AND zone_id = $2 AND status = 'active' AND expires_at > $3`

	var total int
	if err := r.queryRow(ctx, query, eventID, zoneID, now).Scan(&total); err != nil {
		if isInvalidUUID(err) {
			return 0, domain.ErrInvalidID
		}
		return 0, fmt.Errorf("sum active holds: %w", err)
	}
	return total, nil
}

func (r *HoldRepository) SumConfirmed(ctx context.Context, eventID, zoneID string) (int, error) {
	const query = `
SELECT COALESCE(SUM(quantity), 0)
FROM holds
WHERE event_id = $1 AND zone_id = $2 AND status = 'confirmed'`

	var total int
	if err := r.queryRow(ctx, query, eventID, zoneID).Scan(&total); err != nil {
		if isInvalidUUID(err) {
			return 0, domain.ErrInvalidID
		}
		return 0, fmt.Errorf("sum confirmed: %w", err)
	}
	return total, nil
}

func (r *HoldRepository) CreateHold(ctx context.Context, hold domain.Hold) error {
	const stmt = `
INSERT INTO holds (id, event_id, zone_id, quantity, status, expires_at, idempotency_key, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.exec(ctx, stmt,
		hold.ID,
		hold.EventID,
		hold.ZoneID,
		hold.Quantity,
		hold.Status,
		hold.ExpiresAt,
		hold.IdempotencyKey,
		hold.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrIdempotencyConflict
		}
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("create hold: %w", err)
	}
	return nil
}

func (r *HoldRepository) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return r.pool.Exec(ctx, sql, args...)
}

func (r *HoldRepository) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return r.pool.QueryRow(ctx, sql, args...)
}
