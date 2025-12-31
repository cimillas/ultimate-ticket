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

	eventID, _ := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	zone := domain.Zone{
		ID:       "00000000-0000-0000-0000-000000000020",
		EventID:  eventID,
		Name:     "Zone B",
		Capacity: 50,
	}
	if err := repo.CreateZone(ctx, zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}

	zones, err := repo.ListZonesByEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("list zones: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
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

	_, err := repo.ListZonesByEvent(ctx, "not-a-uuid")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}
