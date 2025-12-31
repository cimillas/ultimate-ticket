package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestOrderRepository(t *testing.T) {
	pool := testutil.NewTestPool(t)
	repo := NewOrderRepository(pool)
	testutil.ApplyMigrations(t, context.Background(), pool)

	t.Run("GetHoldForUpdate returns hold or ErrHoldNotFound", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
		holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       2,
			ExpiresAt:      time.Now().Add(5 * time.Minute),
			IdempotencyKey: "idem-hold",
		})

		err := repo.WithTx(ctx, func(txCtx context.Context) error {
			hold, err := repo.GetHoldForUpdate(txCtx, holdID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if hold.ID != holdID || hold.EventID != eventID {
				t.Fatalf("unexpected hold: %+v", hold)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("tx failed: %v", err)
		}

		err = repo.WithTx(ctx, func(txCtx context.Context) error {
			_, err := repo.GetHoldForUpdate(txCtx, "00000000-0000-0000-0000-000000000001")
			if err != domain.ErrHoldNotFound {
				t.Fatalf("expected ErrHoldNotFound, got %v", err)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("tx failed: %v", err)
		}

		_, err = repo.GetHoldForUpdate(ctx, "not-a-uuid")
		if err != domain.ErrInvalidID {
			t.Fatalf("expected ErrInvalidID, got %v", err)
		}
	})

	t.Run("CreateOrder persists and GetOrderByHoldID returns it", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
		holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusConfirmed,
			Quantity:       1,
			ExpiresAt:      time.Now().Add(5 * time.Minute),
			IdempotencyKey: "idem-hold",
		})

		order := domain.Order{
			ID:             "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			HoldID:         holdID,
			IdempotencyKey: "idem-order",
			CreatedAt:      time.Now().UTC(),
		}

		err := repo.WithTx(ctx, func(txCtx context.Context) error {
			return repo.CreateOrder(txCtx, order)
		})
		if err != nil {
			t.Fatalf("create order: %v", err)
		}

		got, err := repo.GetOrderByHoldID(ctx, holdID)
		if err != nil {
			t.Fatalf("get order: %v", err)
		}
		if got == nil {
			t.Fatalf("expected order, got nil")
		}
		if got.HoldID != order.HoldID || got.IdempotencyKey != order.IdempotencyKey {
			t.Fatalf("unexpected order: %+v", got)
		}
	})

	t.Run("UpdateHoldStatus updates status", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
		holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       1,
			ExpiresAt:      time.Now().Add(5 * time.Minute),
			IdempotencyKey: "idem-hold",
		})

		err := repo.WithTx(ctx, func(txCtx context.Context) error {
			return repo.UpdateHoldStatus(txCtx, holdID, domain.HoldStatusConfirmed)
		})
		if err != nil {
			t.Fatalf("update status: %v", err)
		}

		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, holdID).Scan(&status); err != nil {
			t.Fatalf("query status: %v", err)
		}
		if status != string(domain.HoldStatusConfirmed) {
			t.Fatalf("expected status confirmed, got %s", status)
		}
	})
}
