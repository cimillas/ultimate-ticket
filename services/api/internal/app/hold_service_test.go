package app

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestHoldService_CreateHold(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	ttl := 15 * time.Minute

	makeSvc := func(zones []domain.Zone, holds []domain.Hold) (*HoldService, *fakeHoldRepo) {
		repo := newFakeHoldRepo(zones, holds)
		svc := NewHoldService(repo, clock.NewFixed(now), WithHoldTTL(ttl))
		return svc, repo
	}

	t.Run("creates hold when capacity available", func(t *testing.T) {
		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", Quantity: 30, Status: domain.HoldStatusActive, ExpiresAt: now.Add(10 * time.Minute)},
				{EventID: "event-1", ZoneID: "zone-1", Quantity: 20, Status: domain.HoldStatusConfirmed},
			},
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       10,
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if hold.ID == "" {
			t.Fatalf("expected hold ID to be set")
		}
		if hold.Status != domain.HoldStatusActive {
			t.Fatalf("expected status %s, got %s", domain.HoldStatusActive, hold.Status)
		}
		if hold.ExpiresAt != now.Add(ttl) {
			t.Fatalf("expected expires_at %v, got %v", now.Add(ttl), hold.ExpiresAt)
		}
		if len(repo.holds) != 3 {
			t.Fatalf("expected 3 holds in repo, got %d", len(repo.holds))
		}
	})

	t.Run("returns existing hold on idempotency key", func(t *testing.T) {
		existing := domain.Hold{
			ID:              "hold-1",
			EventID:         "event-1",
			ZoneID:          "zone-1",
			Quantity:        5,
			Status:          domain.HoldStatusActive,
			ExpiresAt:       now.Add(ttl),
			IdempotencyKey:  "idem-1",
			CreatedAt:       now,
			IdempotencyHash: "idem-1",
		}

		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 50}},
			[]domain.Hold{existing},
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       5,
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if hold.ID != existing.ID {
			t.Fatalf("expected existing hold ID %s, got %s", existing.ID, hold.ID)
		}
		if len(repo.holds) != 1 {
			t.Fatalf("expected repo holds unchanged, got %d", len(repo.holds))
		}
	})

	t.Run("idempotency conflict on quantity mismatch", func(t *testing.T) {
		existing := domain.Hold{
			ID:             "hold-2",
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       5,
			Status:         domain.HoldStatusActive,
			ExpiresAt:      now.Add(ttl),
			IdempotencyKey: "idem-2",
			CreatedAt:      now,
		}

		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 50}},
			[]domain.Hold{existing},
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       7,
			IdempotencyKey: "idem-2",
		})
		if err != domain.ErrIdempotencyConflict {
			t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
		}
	})

	t.Run("fails when capacity exceeded", func(t *testing.T) {
		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", Quantity: 90, Status: domain.HoldStatusActive, ExpiresAt: now.Add(5 * time.Minute)},
			},
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       20,
			IdempotencyKey: "idem-2",
		})
		if err == nil {
			t.Fatalf("expected error for insufficient capacity")
		}
		if err != domain.ErrInsufficientCapacity {
			t.Fatalf("expected ErrInsufficientCapacity, got %v", err)
		}
		if len(repo.holds) != 1 {
			t.Fatalf("expected holds unchanged on failure, got %d", len(repo.holds))
		}
	})

	t.Run("expired holds free capacity", func(t *testing.T) {
		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", Quantity: 80, Status: domain.HoldStatusActive, ExpiresAt: now.Add(-1 * time.Minute)},
			},
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       50,
			IdempotencyKey: "idem-3",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if hold.Quantity != 50 {
			t.Fatalf("expected quantity 50, got %d", hold.Quantity)
		}
	})

	t.Run("missing idempotency key returns error", func(t *testing.T) {
		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			nil,
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       1,
			IdempotencyKey: "",
		})
		if err != domain.ErrIdempotencyKeyRequired {
			t.Fatalf("expected ErrIdempotencyKeyRequired, got %v", err)
		}
	})
}

type fakeHoldRepo struct {
	zones map[string]domain.Zone
	holds []domain.Hold
}

func newFakeHoldRepo(zones []domain.Zone, holds []domain.Hold) *fakeHoldRepo {
	z := make(map[string]domain.Zone)
	for _, zone := range zones {
		z[zoneKey(zone.EventID, zone.ID)] = zone
	}
	return &fakeHoldRepo{
		zones: z,
		holds: append([]domain.Hold{}, holds...),
	}
}

func (f *fakeHoldRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeHoldRepo) GetZoneForUpdate(_ context.Context, eventID, zoneID string) (domain.Zone, error) {
	zone, ok := f.zones[zoneKey(eventID, zoneID)]
	if !ok {
		return domain.Zone{}, domain.ErrZoneNotFound
	}
	return zone, nil
}

func (f *fakeHoldRepo) FindHoldByIdempotencyKey(_ context.Context, eventID, zoneID, key string) (*domain.Hold, error) {
	for i := range f.holds {
		h := f.holds[i]
		if h.EventID == eventID && h.ZoneID == zoneID && h.IdempotencyKey == key {
			return &h, nil
		}
	}
	return nil, nil
}

func (f *fakeHoldRepo) SumActiveHolds(_ context.Context, eventID, zoneID string, now time.Time) (int, error) {
	total := 0
	for _, h := range f.holds {
		if h.EventID != eventID || h.ZoneID != zoneID {
			continue
		}
		if h.Status != domain.HoldStatusActive {
			continue
		}
		if !h.ExpiresAt.After(now) {
			continue
		}
		total += h.Quantity
	}
	return total, nil
}

func (f *fakeHoldRepo) SumConfirmed(_ context.Context, eventID, zoneID string) (int, error) {
	total := 0
	for _, h := range f.holds {
		if h.EventID != eventID || h.ZoneID != zoneID {
			continue
		}
		if h.Status != domain.HoldStatusConfirmed {
			continue
		}
		total += h.Quantity
	}
	return total, nil
}

func (f *fakeHoldRepo) CreateHold(_ context.Context, hold domain.Hold) error {
	f.holds = append(f.holds, hold)
	return nil
}

func zoneKey(eventID, zoneID string) string {
	return eventID + "|" + zoneID
}
