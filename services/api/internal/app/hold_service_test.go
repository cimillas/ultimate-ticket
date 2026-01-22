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

	makeSvc := func(zones []domain.Zone, holds []domain.Hold, eventStarts map[string]time.Time) (*HoldService, *fakeHoldRepo) {
		repo := newFakeHoldRepo(zones, holds, eventStarts, nil, now.Add(1*time.Hour))
		svc := NewHoldService(repo, clock.NewFixed(now), WithHoldTTL(ttl))
		return svc, repo
	}

	t.Run("creates hold when capacity available", func(t *testing.T) {
		userID := "user-1"
		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", UserID: userID, Quantity: 30, Status: domain.HoldStatusActive, ExpiresAt: now.Add(10 * time.Minute)},
				{EventID: "event-1", ZoneID: "zone-1", UserID: userID, Quantity: 20, Status: domain.HoldStatusConfirmed},
			},
			nil,
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
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
		userID := "user-1"
		existing := domain.Hold{
			ID:              "hold-1",
			EventID:         "event-1",
			ZoneID:          "zone-1",
			UserID:          userID,
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
			nil,
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
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

	t.Run("idempotency scoped per user", func(t *testing.T) {
		existing := domain.Hold{
			ID:             "hold-1",
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         "user-1",
			Quantity:       5,
			Status:         domain.HoldStatusActive,
			ExpiresAt:      now.Add(ttl),
			IdempotencyKey: "idem-1",
		}

		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 50}},
			[]domain.Hold{existing},
			nil,
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         "user-2",
			Quantity:       5,
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if hold.ID == existing.ID {
			t.Fatalf("expected a new hold for a different user")
		}
		if len(repo.holds) != 2 {
			t.Fatalf("expected 2 holds in repo, got %d", len(repo.holds))
		}
	})

	t.Run("idempotency conflict on quantity mismatch", func(t *testing.T) {
		userID := "user-1"
		existing := domain.Hold{
			ID:             "hold-2",
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
			Quantity:       5,
			Status:         domain.HoldStatusActive,
			ExpiresAt:      now.Add(ttl),
			IdempotencyKey: "idem-2",
			CreatedAt:      now,
		}

		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 50}},
			[]domain.Hold{existing},
			nil,
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
			Quantity:       7,
			IdempotencyKey: "idem-2",
		})
		if err != domain.ErrIdempotencyConflict {
			t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
		}
	})

	t.Run("fails when capacity exceeded", func(t *testing.T) {
		userID := "user-1"
		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", UserID: userID, Quantity: 90, Status: domain.HoldStatusActive, ExpiresAt: now.Add(5 * time.Minute)},
			},
			nil,
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
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
		userID := "user-1"
		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			[]domain.Hold{
				{EventID: "event-1", ZoneID: "zone-1", UserID: userID, Quantity: 80, Status: domain.HoldStatusActive, ExpiresAt: now.Add(-1 * time.Minute)},
			},
			nil,
		)

		hold, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
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
		userID := "user-1"
		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			nil,
			nil,
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
			Quantity:       1,
			IdempotencyKey: "",
		})
		if err != domain.ErrIdempotencyKeyRequired {
			t.Fatalf("expected ErrIdempotencyKeyRequired, got %v", err)
		}
	})

	t.Run("missing user returns error", func(t *testing.T) {
		svc, _ := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 100}},
			nil,
			nil,
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			Quantity:       1,
			IdempotencyKey: "idem-user",
		})
		if err != domain.ErrUnauthorized {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("rejects holds after event start", func(t *testing.T) {
		userID := "user-1"
		svc, repo := makeSvc(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 10}},
			nil,
			map[string]time.Time{"event-1": now.Add(-1 * time.Minute)},
		)

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
			Quantity:       1,
			IdempotencyKey: "idem-started",
		})
		if err != domain.ErrEventClosed {
			t.Fatalf("expected ErrEventClosed, got %v", err)
		}
		if len(repo.holds) != 0 {
			t.Fatalf("expected no holds created, got %d", len(repo.holds))
		}
		if repo.updatedEvents["event-1"] != domain.EventStatusClosed {
			t.Fatalf("expected event to be closed, got %s", repo.updatedEvents["event-1"])
		}
	})

	t.Run("rejects holds when event is cancelled", func(t *testing.T) {
		userID := "user-1"
		repo := newFakeHoldRepo(
			[]domain.Zone{{ID: "zone-1", EventID: "event-1", Capacity: 10}},
			nil,
			nil,
			map[string]domain.EventStatus{"event-1": domain.EventStatusCancelled},
			now.Add(1*time.Hour),
		)
		svc := NewHoldService(repo, clock.NewFixed(now), WithHoldTTL(ttl))

		_, err := svc.CreateHold(context.Background(), CreateHoldInput{
			EventID:        "event-1",
			ZoneID:         "zone-1",
			UserID:         userID,
			Quantity:       1,
			IdempotencyKey: "idem-cancelled",
		})
		if err != domain.ErrEventCancelled {
			t.Fatalf("expected ErrEventCancelled, got %v", err)
		}
		if len(repo.holds) != 0 {
			t.Fatalf("expected no holds created, got %d", len(repo.holds))
		}
	})
}

func TestHoldService_ListActiveHoldsByUser(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)

	t.Run("returns active, unexpired holds for the user", func(t *testing.T) {
		userID := "user-1"
		holds := []domain.Hold{
			{ID: "hold-1", UserID: userID, Status: domain.HoldStatusActive, ExpiresAt: now.Add(5 * time.Minute)},
			{ID: "hold-2", UserID: userID, Status: domain.HoldStatusActive, ExpiresAt: now.Add(-1 * time.Minute)},
			{ID: "hold-3", UserID: userID, Status: domain.HoldStatusConfirmed, ExpiresAt: now.Add(5 * time.Minute)},
			{ID: "hold-4", UserID: "user-2", Status: domain.HoldStatusActive, ExpiresAt: now.Add(5 * time.Minute)},
		}
		repo := newFakeHoldRepo(nil, holds, nil, nil, now.Add(1*time.Hour))
		svc := NewHoldService(repo, clock.NewFixed(now))

		active, err := svc.ListActiveHoldsByUser(context.Background(), userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(active) != 1 {
			t.Fatalf("expected 1 hold, got %d", len(active))
		}
		if active[0].ID != "hold-1" {
			t.Fatalf("expected hold-1, got %s", active[0].ID)
		}
	})

	t.Run("missing user returns error", func(t *testing.T) {
		repo := newFakeHoldRepo(nil, nil, nil, nil, now.Add(1*time.Hour))
		svc := NewHoldService(repo, clock.NewFixed(now))

		_, err := svc.ListActiveHoldsByUser(context.Background(), "")
		if err != domain.ErrUnauthorized {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

type fakeHoldRepo struct {
	zones             map[string]domain.Zone
	holds             []domain.Hold
	eventStarts       map[string]time.Time
	eventStatuses     map[string]domain.EventStatus
	defaultEventStart time.Time
	updatedEvents     map[string]domain.EventStatus
}

func newFakeHoldRepo(zones []domain.Zone, holds []domain.Hold, eventStarts map[string]time.Time, eventStatuses map[string]domain.EventStatus, defaultEventStart time.Time) *fakeHoldRepo {
	z := make(map[string]domain.Zone)
	for _, zone := range zones {
		z[zoneKey(zone.EventID, zone.ID)] = zone
	}
	return &fakeHoldRepo{
		zones:             z,
		holds:             append([]domain.Hold{}, holds...),
		eventStarts:       eventStarts,
		eventStatuses:     eventStatuses,
		defaultEventStart: defaultEventStart,
	}
}

func (f *fakeHoldRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeHoldRepo) GetZoneForUpdate(_ context.Context, eventID, zoneID string) (domain.Zone, time.Time, domain.EventStatus, error) {
	zone, ok := f.zones[zoneKey(eventID, zoneID)]
	if !ok {
		return domain.Zone{}, time.Time{}, "", domain.ErrZoneNotFound
	}
	startsAt := f.defaultEventStart
	if f.eventStarts != nil {
		if override, ok := f.eventStarts[eventID]; ok {
			startsAt = override
		}
	}
	status := domain.EventStatusActive
	if f.eventStatuses != nil {
		if override, ok := f.eventStatuses[eventID]; ok {
			status = override
		}
	}
	return zone, startsAt, status, nil
}

func (f *fakeHoldRepo) FindHoldByIdempotencyKey(_ context.Context, eventID, zoneID, userID, key string) (*domain.Hold, error) {
	for i := range f.holds {
		h := f.holds[i]
		if h.EventID == eventID && h.ZoneID == zoneID && h.UserID == userID && h.IdempotencyKey == key {
			return &h, nil
		}
	}
	return nil, nil
}

func (f *fakeHoldRepo) ListActiveHoldsByUser(_ context.Context, userID string, now time.Time) ([]domain.Hold, error) {
	var result []domain.Hold
	for _, h := range f.holds {
		if h.UserID != userID {
			continue
		}
		if h.Status != domain.HoldStatusActive {
			continue
		}
		if !h.ExpiresAt.After(now) {
			continue
		}
		result = append(result, h)
	}
	return result, nil
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

func (f *fakeHoldRepo) UpdateEventStatus(_ context.Context, eventID string, status domain.EventStatus) error {
	if f.updatedEvents == nil {
		f.updatedEvents = make(map[string]domain.EventStatus)
	}
	f.updatedEvents[eventID] = status
	if f.eventStatuses == nil {
		f.eventStatuses = make(map[string]domain.EventStatus)
	}
	f.eventStatuses[eventID] = status
	return nil
}

func zoneKey(eventID, zoneID string) string {
	return eventID + "|" + zoneID
}
