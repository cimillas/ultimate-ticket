package app

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestOrderService_ConfirmHold(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	defaultEventStart := now.Add(1 * time.Hour)

	t.Run("creates order for active hold", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-1": {
				ID:        "hold-1",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, nil, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		res, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-1",
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !res.Created {
			t.Fatalf("expected Created=true")
		}
		if res.Order.ID == "" {
			t.Fatalf("expected order ID to be set")
		}
		if res.Order.HoldID != "hold-1" {
			t.Fatalf("expected hold_id hold-1, got %s", res.Order.HoldID)
		}
		if res.Order.IdempotencyKey != "idem-1" {
			t.Fatalf("expected idempotency key idem-1, got %s", res.Order.IdempotencyKey)
		}

		hold := repo.holds["hold-1"]
		if hold.Status != domain.HoldStatusConfirmed {
			t.Fatalf("expected hold status confirmed, got %s", hold.Status)
		}
		if _, ok := repo.orders["hold-1"]; !ok {
			t.Fatalf("expected order persisted")
		}
	})

	t.Run("idempotent confirm returns existing order", func(t *testing.T) {
		existing := domain.Order{
			ID:             "order-1",
			HoldID:         "hold-2",
			IdempotencyKey: "idem-1",
			CreatedAt:      now.Add(-1 * time.Minute),
		}
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-2": {
				ID:        "hold-2",
				EventID:   "event-1",
				Status:    domain.HoldStatusConfirmed,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, nil, defaultEventStart)
		repo.orders["hold-2"] = existing

		svc := NewOrderService(repo, clock.NewFixed(now))

		res, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-2",
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.Created {
			t.Fatalf("expected Created=false")
		}
		if res.Order.ID != existing.ID {
			t.Fatalf("expected existing order ID %s, got %s", existing.ID, res.Order.ID)
		}
	})

	t.Run("different idempotency key after confirmed returns error", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-3": {
				ID:        "hold-3",
				EventID:   "event-1",
				Status:    domain.HoldStatusConfirmed,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, nil, defaultEventStart)
		repo.orders["hold-3"] = domain.Order{
			ID:             "order-3",
			HoldID:         "hold-3",
			IdempotencyKey: "idem-1",
			CreatedAt:      now,
		}

		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-3",
			IdempotencyKey: "idem-2",
		})
		if err != domain.ErrHoldAlreadyConfirmed {
			t.Fatalf("expected ErrHoldAlreadyConfirmed, got %v", err)
		}
	})

	t.Run("expired hold returns error", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-4": {
				ID:        "hold-4",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(-1 * time.Minute),
			},
		}, nil, nil, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-4",
			IdempotencyKey: "idem-1",
		})
		if err != domain.ErrHoldExpired {
			t.Fatalf("expected ErrHoldExpired, got %v", err)
		}
	})

	t.Run("missing idempotency key returns error", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-5": {
				ID:        "hold-5",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, nil, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-5",
			IdempotencyKey: "",
		})
		if err != domain.ErrIdempotencyKeyRequired {
			t.Fatalf("expected ErrIdempotencyKeyRequired, got %v", err)
		}
	})

	t.Run("missing hold returns error", func(t *testing.T) {
		repo := newFakeOrderRepo(nil, nil, nil, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "missing",
			IdempotencyKey: "idem-1",
		})
		if err != domain.ErrHoldNotFound {
			t.Fatalf("expected ErrHoldNotFound, got %v", err)
		}
	})

	t.Run("idempotent on create conflict when order exists", func(t *testing.T) {
		repo := &raceOrderRepo{
			hold: domain.Hold{
				ID:        "hold-6",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
			startsAt: defaultEventStart,
			order: domain.Order{
				ID:             "order-6",
				HoldID:         "hold-6",
				IdempotencyKey: "idem-1",
				CreatedAt:      now,
			},
		}
		svc := NewOrderService(repo, clock.NewFixed(now))

		res, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-6",
			IdempotencyKey: "idem-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.Created {
			t.Fatalf("expected Created=false")
		}
		if res.Order.ID != "order-6" {
			t.Fatalf("expected order-6, got %s", res.Order.ID)
		}
	})

	t.Run("event started cancels hold", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-7": {
				ID:        "hold-7",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, map[string]time.Time{"event-1": now.Add(-1 * time.Minute)}, nil, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-7",
			IdempotencyKey: "idem-started",
		})
		if err != domain.ErrEventClosed {
			t.Fatalf("expected ErrEventClosed, got %v", err)
		}

		hold := repo.holds["hold-7"]
		if hold.Status != domain.HoldStatusInvalid {
			t.Fatalf("expected hold status invalid, got %s", hold.Status)
		}
		if _, ok := repo.orders["hold-7"]; ok {
			t.Fatalf("expected no order created")
		}
		if repo.updatedEvents["event-1"] != domain.EventStatusClosed {
			t.Fatalf("expected event to be closed, got %s", repo.updatedEvents["event-1"])
		}
	})

	t.Run("event cancelled blocks confirmation", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-8": {
				ID:        "hold-8",
				EventID:   "event-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, map[string]domain.EventStatus{"event-1": domain.EventStatusCancelled}, defaultEventStart)
		svc := NewOrderService(repo, clock.NewFixed(now))

		_, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-8",
			IdempotencyKey: "idem-cancelled",
		})
		if err != domain.ErrEventCancelled {
			t.Fatalf("expected ErrEventCancelled, got %v", err)
		}

		hold := repo.holds["hold-8"]
		if hold.Status != domain.HoldStatusInvalid {
			t.Fatalf("expected hold status invalid, got %s", hold.Status)
		}
		if _, ok := repo.orders["hold-8"]; ok {
			t.Fatalf("expected no order created")
		}
	})

	t.Run("idempotent confirm returns order even when event cancelled", func(t *testing.T) {
		existing := domain.Order{
			ID:             "order-9",
			HoldID:         "hold-9",
			IdempotencyKey: "idem-9",
			Status:         domain.OrderStatusRefunded,
			CreatedAt:      now.Add(-2 * time.Minute),
		}
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-9": {
				ID:        "hold-9",
				EventID:   "event-1",
				Status:    domain.HoldStatusConfirmed,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		}, nil, map[string]domain.EventStatus{"event-1": domain.EventStatusCancelled}, defaultEventStart)
		repo.orders["hold-9"] = existing

		svc := NewOrderService(repo, clock.NewFixed(now))

		res, err := svc.ConfirmHold(context.Background(), ConfirmHoldInput{
			HoldID:         "hold-9",
			IdempotencyKey: "idem-9",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.Order.ID != existing.ID {
			t.Fatalf("expected order %s, got %s", existing.ID, res.Order.ID)
		}
		if res.Order.Status != domain.OrderStatusRefunded {
			t.Fatalf("expected refunded order, got %s", res.Order.Status)
		}
	})
}

type fakeOrderRepo struct {
	holds             map[string]domain.Hold
	orders            map[string]domain.Order
	eventStarts       map[string]time.Time
	eventStatuses     map[string]domain.EventStatus
	defaultEventStart time.Time
	updatedEvents     map[string]domain.EventStatus
}

func newFakeOrderRepo(holds map[string]domain.Hold, eventStarts map[string]time.Time, eventStatuses map[string]domain.EventStatus, defaultEventStart time.Time) *fakeOrderRepo {
	if holds == nil {
		holds = make(map[string]domain.Hold)
	}
	return &fakeOrderRepo{
		holds:             holds,
		orders:            make(map[string]domain.Order),
		eventStarts:       eventStarts,
		eventStatuses:     eventStatuses,
		defaultEventStart: defaultEventStart,
	}
}

func (f *fakeOrderRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeOrderRepo) GetHoldForUpdate(_ context.Context, holdID string) (domain.Hold, time.Time, domain.EventStatus, error) {
	hold, ok := f.holds[holdID]
	if !ok {
		return domain.Hold{}, time.Time{}, "", domain.ErrHoldNotFound
	}
	startsAt := f.defaultEventStart
	if f.eventStarts != nil {
		if override, ok := f.eventStarts[hold.EventID]; ok {
			startsAt = override
		}
	}
	status := domain.EventStatusActive
	if f.eventStatuses != nil {
		if override, ok := f.eventStatuses[hold.EventID]; ok {
			status = override
		}
	}
	return hold, startsAt, status, nil
}

func (f *fakeOrderRepo) GetOrderByHoldID(_ context.Context, holdID string) (*domain.Order, error) {
	order, ok := f.orders[holdID]
	if !ok {
		return nil, nil
	}
	copy := order
	return &copy, nil
}

func (f *fakeOrderRepo) CreateOrder(_ context.Context, order domain.Order) error {
	if _, exists := f.orders[order.HoldID]; exists {
		return domain.ErrHoldAlreadyConfirmed
	}
	f.orders[order.HoldID] = order
	return nil
}

func (f *fakeOrderRepo) UpdateHoldStatus(_ context.Context, holdID string, status domain.HoldStatus) error {
	hold, ok := f.holds[holdID]
	if !ok {
		return domain.ErrHoldNotFound
	}
	hold.Status = status
	f.holds[holdID] = hold
	return nil
}

func (f *fakeOrderRepo) UpdateEventStatus(_ context.Context, eventID string, status domain.EventStatus) error {
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

type raceOrderRepo struct {
	hold     domain.Hold
	startsAt time.Time
	order    domain.Order
	looked   bool
}

func (r *raceOrderRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (r *raceOrderRepo) GetHoldForUpdate(_ context.Context, holdID string) (domain.Hold, time.Time, domain.EventStatus, error) {
	if r.hold.ID != holdID {
		return domain.Hold{}, time.Time{}, "", domain.ErrHoldNotFound
	}
	return r.hold, r.startsAt, domain.EventStatusActive, nil
}

func (r *raceOrderRepo) GetOrderByHoldID(_ context.Context, holdID string) (*domain.Order, error) {
	if r.looked {
		return &r.order, nil
	}
	r.looked = true
	return nil, nil
}

func (r *raceOrderRepo) CreateOrder(_ context.Context, _ domain.Order) error {
	return domain.ErrHoldAlreadyConfirmed
}

func (r *raceOrderRepo) UpdateHoldStatus(_ context.Context, _ string, _ domain.HoldStatus) error {
	return nil
}

func (r *raceOrderRepo) UpdateEventStatus(_ context.Context, _ string, _ domain.EventStatus) error {
	return nil
}
