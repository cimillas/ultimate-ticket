package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestHoldRepository(t *testing.T) {
	pool := testutil.NewTestPool(t)
	repo := NewHoldRepository(pool)
	testutil.ApplyMigrations(t, context.Background(), pool)

	t.Run("GetZoneForUpdate returns zone and ErrZoneNotFound", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)

		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

		err := repo.WithTx(ctx, func(txCtx context.Context) error {
			zone, err := repo.GetZoneForUpdate(txCtx, eventID, zoneID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if zone.ID != zoneID || zone.EventID != eventID || zone.Capacity != 100 {
				t.Fatalf("unexpected zone: %+v", zone)
			}

			missingZoneID := "00000000-0000-0000-0000-000000000001"
			_, err = repo.GetZoneForUpdate(txCtx, eventID, missingZoneID)
			if err != domain.ErrZoneNotFound {
				t.Fatalf("expected ErrZoneNotFound, got %v", err)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("tx failed: %v", err)
		}

		_, err = repo.GetZoneForUpdate(ctx, eventID, "not-a-uuid")
		if err != domain.ErrInvalidID {
			t.Fatalf("expected ErrInvalidID, got %v", err)
		}
	})

	t.Run("FindHoldByIdempotencyKey returns existing hold", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 50)

		holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       5,
			ExpiresAt:      time.Now().Add(10 * time.Minute).UTC(),
			IdempotencyKey: "idem-1",
		})

		h, err := repo.FindHoldByIdempotencyKey(ctx, eventID, zoneID, "idem-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if h == nil || h.ID != holdID || h.IdempotencyKey != "idem-1" {
			t.Fatalf("unexpected hold: %+v", h)
		}

		h, err = repo.FindHoldByIdempotencyKey(ctx, eventID, zoneID, "missing")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if h != nil {
			t.Fatalf("expected nil, got %+v", h)
		}
	})

	t.Run("SumActiveHolds excludes expired", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
		now := time.Now().UTC()

		testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       30,
			ExpiresAt:      now.Add(5 * time.Minute),
			IdempotencyKey: "a",
		})
		testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       20,
			ExpiresAt:      now.Add(-1 * time.Minute), // expired
			IdempotencyKey: "b",
		})

		total, err := repo.SumActiveHolds(ctx, eventID, zoneID, now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 30 {
			t.Fatalf("expected active sum 30, got %d", total)
		}
	})

	t.Run("SumConfirmed sums confirmed only", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

		testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusConfirmed,
			Quantity:       10,
			ExpiresAt:      time.Now().Add(5 * time.Minute),
			IdempotencyKey: "c",
		})
		testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
			Status:         domain.HoldStatusActive,
			Quantity:       5,
			ExpiresAt:      time.Now().Add(5 * time.Minute),
			IdempotencyKey: "d",
		})

		total, err := repo.SumConfirmed(ctx, eventID, zoneID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if total != 10 {
			t.Fatalf("expected confirmed sum 10, got %d", total)
		}
	})

	t.Run("CreateHold inserts row", func(t *testing.T) {
		ctx := context.Background()
		testutil.TruncateAll(t, ctx, pool)
		eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
		now := time.Now().UTC()

		hold := domain.Hold{
			ID:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			EventID:        eventID,
			ZoneID:         zoneID,
			Quantity:       5,
			Status:         domain.HoldStatusActive,
			ExpiresAt:      now.Add(10 * time.Minute),
			IdempotencyKey: "idem-create",
			CreatedAt:      now,
		}
		if err := repo.CreateHold(ctx, hold); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		var count int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM holds WHERE id = $1", hold.ID).Scan(&count); err != nil {
			t.Fatalf("query count: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected hold persisted, got count %d", count)
		}
	})
}
