package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestAdminRepository_CreateAndListEvents(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	event := domain.Event{
		ID:       "00000000-0000-0000-0000-000000000010",
		Name:     "Concert",
		StartsAt: time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC),
	}
	if err := repo.CreateEvent(ctx, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	events, err := repo.ListEvents(ctx)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != event.ID || events[0].Name != event.Name {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestAdminRepository_CreateAndListZones(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	now := time.Now().UTC()
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       10,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "active-1",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       20,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "confirmed-1",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       5,
		ExpiresAt:      now.Add(-10 * time.Minute),
		IdempotencyKey: "expired-1",
	})

	zone := domain.Zone{
		ID:       "00000000-0000-0000-0000-000000000020",
		EventID:  eventID,
		Name:     "Zone B",
		Capacity: 50,
	}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}

	zones, err := repo.ListZonesByEvent(ctx, eventID, time.Now().UTC())
	if err != nil {
		t.Fatalf("list zones: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	for _, z := range zones {
		if z.ID == zoneID && z.Available != 70 {
			t.Fatalf("expected available 70 for zone A, got %d", z.Available)
		}
		if z.ID == zone.ID && z.Available != zone.Capacity {
			t.Fatalf("expected available %d for new zone, got %d", zone.Capacity, z.Available)
		}
	}
}

func TestAdminRepository_CreateZone_InvalidEvent(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	zone := domain.Zone{
		ID:       "00000000-0000-0000-0000-000000000030",
		EventID:  "00000000-0000-0000-0000-000000000031",
		Name:     "Zone A",
		Capacity: 10,
	}
	if err := repo.CreateZone(ctx, zone); err != domain.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}

	_, err := repo.ListZonesByEvent(ctx, "not-a-uuid", time.Now().UTC())
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestAdminRepository_ListActiveHoldsByZone(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	now := time.Now().UTC()

	activeHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      now.Add(5 * time.Minute),
		IdempotencyKey: "active-1",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      now.Add(-5 * time.Minute),
		IdempotencyKey: "expired-1",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      now.Add(5 * time.Minute),
		IdempotencyKey: "confirmed-1",
	})

	holds, err := repo.ListActiveHoldsByZone(ctx, eventID, zoneID, now)
	if err != nil {
		t.Fatalf("list active holds: %v", err)
	}
	if len(holds) != 1 {
		t.Fatalf("expected 1 active hold, got %d", len(holds))
	}
	if holds[0].ID != activeHoldID {
		t.Fatalf("expected hold %s, got %s", activeHoldID, holds[0].ID)
	}
}

func TestAdminRepository_ListActiveHoldsByZone_InvalidIDs(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	now := time.Now().UTC()

	_, err := repo.ListActiveHoldsByZone(ctx, "not-a-uuid", zoneID, now)
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = repo.ListActiveHoldsByZone(ctx, "00000000-0000-0000-0000-000000000099", zoneID, now)
	if err != domain.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}

	_, err = repo.ListActiveHoldsByZone(ctx, eventID, "00000000-0000-0000-0000-000000000100", now)
	if err != domain.ErrZoneNotFound {
		t.Fatalf("expected ErrZoneNotFound, got %v", err)
	}
}

func TestAdminRepository_ListOrdersByZone(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		IdempotencyKey: "idem-hold",
	})
	orderID := "00000000-0000-0000-0000-000000000070"
	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, created_at, status)
VALUES ($1, $2, $3, $4, $5)`,
		orderID, holdID, "idem-order", time.Now().UTC(), domain.OrderStatusConfirmed,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	refundedHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		IdempotencyKey: "idem-refund",
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, created_at, status)
VALUES ($1, $2, $3, $4, $5)`,
		"00000000-0000-0000-0000-000000000071", refundedHoldID, "idem-refund", time.Now().UTC(), domain.OrderStatusRefunded,
	); err != nil {
		t.Fatalf("insert refunded order: %v", err)
	}

	orders, err := repo.ListOrdersByZone(ctx, eventID, zoneID)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].ID != orderID {
		t.Fatalf("expected order %s, got %s", orderID, orders[0].ID)
	}
}

func TestAdminRepository_CancelEvent(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	activeHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		IdempotencyKey: "active-1",
	})
	confirmedHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		IdempotencyKey: "confirmed-1",
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, created_at, status)
VALUES ($1, $2, $3, $4, $5)`,
		"00000000-0000-0000-0000-000000000080", confirmedHoldID, "idem-order", time.Now().UTC(), domain.OrderStatusConfirmed,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	now := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)
	cancelled, err := repo.CancelEvent(ctx, eventID, now)
	if err != nil {
		t.Fatalf("cancel event: %v", err)
	}
	if cancelled.Status != domain.EventStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}
	if cancelled.CancelledAt == nil || !cancelled.CancelledAt.Equal(now) {
		t.Fatalf("expected cancelled_at %v, got %v", now, cancelled.CancelledAt)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, activeHoldID).Scan(&status); err != nil {
		t.Fatalf("query active hold status: %v", err)
	}
	if status != string(domain.HoldStatusInvalid) {
		t.Fatalf("expected active hold invalid, got %s", status)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, confirmedHoldID).Scan(&status); err != nil {
		t.Fatalf("query confirmed hold status: %v", err)
	}
	if status != string(domain.HoldStatusInvalid) {
		t.Fatalf("expected confirmed hold invalid, got %s", status)
	}

	var orderStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE hold_id = $1`, confirmedHoldID).Scan(&orderStatus); err != nil {
		t.Fatalf("query order status: %v", err)
	}
	if orderStatus != string(domain.OrderStatusRefunded) {
		t.Fatalf("expected refunded order, got %s", orderStatus)
	}
}

func TestAdminRepository_ListOrdersByZone_InvalidIDs(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := NewAdminRepository(pool)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

	_, err := repo.ListOrdersByZone(ctx, "not-a-uuid", zoneID)
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = repo.ListOrdersByZone(ctx, "00000000-0000-0000-0000-000000000099", zoneID)
	if err != domain.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}

	_, err = repo.ListOrdersByZone(ctx, eventID, "00000000-0000-0000-0000-000000000100")
	if err != domain.ErrZoneNotFound {
		t.Fatalf("expected ErrZoneNotFound, got %v", err)
	}
}
