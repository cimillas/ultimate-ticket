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

	t.Run("creates order for active hold", func(t *testing.T) {
		repo := newFakeOrderRepo(map[string]domain.Hold{
			"hold-1": {
				ID:        "hold-1",
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		})
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
				Status:    domain.HoldStatusConfirmed,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		})
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
				Status:    domain.HoldStatusConfirmed,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		})
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
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(-1 * time.Minute),
			},
		})
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
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
		})
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
		repo := newFakeOrderRepo(nil)
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
				Status:    domain.HoldStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
			},
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
}

type fakeOrderRepo struct {
	holds  map[string]domain.Hold
	orders map[string]domain.Order
}

func newFakeOrderRepo(holds map[string]domain.Hold) *fakeOrderRepo {
	if holds == nil {
		holds = make(map[string]domain.Hold)
	}
	return &fakeOrderRepo{
		holds:  holds,
		orders: make(map[string]domain.Order),
	}
}

func (f *fakeOrderRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeOrderRepo) GetHoldForUpdate(_ context.Context, holdID string) (domain.Hold, error) {
	hold, ok := f.holds[holdID]
	if !ok {
		return domain.Hold{}, domain.ErrHoldNotFound
	}
	return hold, nil
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

type raceOrderRepo struct {
	hold   domain.Hold
	order  domain.Order
	looked bool
}

func (r *raceOrderRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (r *raceOrderRepo) GetHoldForUpdate(_ context.Context, holdID string) (domain.Hold, error) {
	if r.hold.ID != holdID {
		return domain.Hold{}, domain.ErrHoldNotFound
	}
	return r.hold, nil
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
