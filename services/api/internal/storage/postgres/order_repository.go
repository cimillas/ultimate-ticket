package postgres

import (
	"context"
	"fmt"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

func (r *OrderRepository) GetHoldForUpdate(ctx context.Context, holdID string) (domain.Hold, error) {
	const query = `
SELECT id, event_id, zone_id, quantity, status, expires_at
FROM holds
WHERE id = $1
FOR UPDATE`

	var h domain.Hold
	var status string
	err := r.queryRow(ctx, query, holdID).
		Scan(&h.ID, &h.EventID, &h.ZoneID, &h.Quantity, &status, &h.ExpiresAt)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.Hold{}, domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return domain.Hold{}, domain.ErrHoldNotFound
		}
		return domain.Hold{}, fmt.Errorf("get hold: %w", err)
	}
	h.Status = domain.HoldStatus(status)
	return h, nil
}

func (r *OrderRepository) GetOrderByHoldID(ctx context.Context, holdID string) (*domain.Order, error) {
	const query = `SELECT id, hold_id, idempotency_key, created_at FROM orders WHERE hold_id = $1`

	var o domain.Order
	err := r.queryRow(ctx, query, holdID).
		Scan(&o.ID, &o.HoldID, &o.IdempotencyKey, &o.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order domain.Order) error {
	const stmt = `
INSERT INTO orders (id, hold_id, idempotency_key, created_at)
VALUES ($1, $2, $3, $4)`

	_, err := r.exec(ctx, stmt, order.ID, order.HoldID, order.IdempotencyKey, order.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrHoldAlreadyConfirmed
		}
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error {
	const stmt = `UPDATE holds SET status = $2 WHERE id = $1`

	tag, err := r.exec(ctx, stmt, holdID, status)
	if err != nil {
		return fmt.Errorf("update hold status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrHoldNotFound
	}
	return nil
}

func (r *OrderRepository) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return r.pool.Exec(ctx, sql, args...)
}

func (r *OrderRepository) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return r.pool.QueryRow(ctx, sql, args...)
}
