package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/cimillas/ultimate-ticket/services/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTestDBURL       = "postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable"
	testDBLockID     int64 = 801234568
)

func NewTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDBURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	cfg.MaxConns = 4

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping Postgres integration tests: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	lockTestDB(t, pool)

	return pool
}

func ApplyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}
}

func TruncateAll(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `TRUNCATE orders, holds, zones, events RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func InsertEventAndZone(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string, capacity int) (eventID, zoneID string) {
	t.Helper()
	if err := pool.QueryRow(ctx,
		`INSERT INTO events (name, starts_at) VALUES ($1, NOW()) RETURNING id`,
		name,
	).Scan(&eventID); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO zones (event_id, name, capacity) VALUES ($1, $2, $3) RETURNING id`,
		eventID, "Zone A", capacity,
	).Scan(&zoneID); err != nil {
		t.Fatalf("insert zone: %v", err)
	}
	return
}

func InsertHold(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventID, zoneID string, hold domain.Hold) string {
	t.Helper()
	var id string
	err := pool.QueryRow(ctx, `
INSERT INTO holds (event_id, zone_id, quantity, status, expires_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`,
		eventID, zoneID, hold.Quantity, hold.Status, hold.ExpiresAt, hold.IdempotencyKey,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert hold: %v", err)
	}
	return id
}

func lockTestDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock conn: %v", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, testDBLockID); err != nil {
		conn.Release()
		t.Fatalf("acquire test lock: %v", err)
	}

	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, testDBLockID)
		conn.Release()
	})
}
