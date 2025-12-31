package migrations_test

import (
	"context"
	"testing"

	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
	"github.com/cimillas/ultimate-ticket/services/api/migrations"
)

func TestApply_RecordsMigrations(t *testing.T) {
	pool := testutil.NewTestPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations`); err != nil {
		t.Fatalf("drop schema_migrations: %v", err)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected at least 2 migrations, got %d", count)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("re-apply migrations: %v", err)
	}

	var count2 int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count2); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count2 != count {
		t.Fatalf("expected migration count unchanged, got %d vs %d", count2, count)
	}
}
