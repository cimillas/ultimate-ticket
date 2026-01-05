package app

import (
	"context"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type OrderRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	GetHoldForUpdate(ctx context.Context, holdID string) (domain.Hold, time.Time, domain.EventStatus, error)
	GetOrderByHoldID(ctx context.Context, holdID string) (*domain.Order, error)
	CreateOrder(ctx context.Context, order domain.Order) error
	UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error
	UpdateEventStatus(ctx context.Context, eventID string, status domain.EventStatus) error
}

type OrderService struct {
	repo  OrderRepository
	clock clock.Clock
}

func NewOrderService(repo OrderRepository, clk clock.Clock) *OrderService {
	return &OrderService{
		repo:  repo,
		clock: clk,
	}
}

type ConfirmHoldInput struct {
	HoldID         string
	IdempotencyKey string
}

type ConfirmHoldResult struct {
	Order   domain.Order
	Created bool
}

func (s *OrderService) ConfirmHold(ctx context.Context, in ConfirmHoldInput) (ConfirmHoldResult, error) {
	if in.IdempotencyKey == "" {
		return ConfirmHoldResult{}, domain.ErrIdempotencyKeyRequired
	}

	now := s.clock.Now()
	var result ConfirmHoldResult
	var eventErr error

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		hold, startsAt, eventStatus, err := s.repo.GetHoldForUpdate(txCtx, in.HoldID)
		if err != nil {
			return err
		}

		existing, err := s.repo.GetOrderByHoldID(txCtx, in.HoldID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.IdempotencyKey == in.IdempotencyKey {
				result = ConfirmHoldResult{Order: *existing, Created: false}
				return nil
			}
			return domain.ErrHoldAlreadyConfirmed
		}

		if eventStatus == domain.EventStatusCancelled {
			if hold.Status == domain.HoldStatusActive {
				if err := s.repo.UpdateHoldStatus(txCtx, in.HoldID, domain.HoldStatusInvalid); err != nil {
					return err
				}
			}
			eventErr = domain.ErrEventCancelled
			return nil
		}
		if eventStatus == domain.EventStatusClosed {
			if hold.Status == domain.HoldStatusActive {
				if err := s.repo.UpdateHoldStatus(txCtx, in.HoldID, domain.HoldStatusInvalid); err != nil {
					return err
				}
			}
			eventErr = domain.ErrEventClosed
			return nil
		}

		if hold.Status == domain.HoldStatusInvalid {
			return domain.ErrHoldInvalid
		}
		if hold.Status == domain.HoldStatusConfirmed {
			return domain.ErrHoldAlreadyConfirmed
		}
		if !startsAt.After(now) {
			if hold.Status == domain.HoldStatusActive {
				if err := s.repo.UpdateHoldStatus(txCtx, in.HoldID, domain.HoldStatusInvalid); err != nil {
					return err
				}
			}
			if eventStatus == "" || eventStatus == domain.EventStatusActive {
				if err := s.repo.UpdateEventStatus(txCtx, hold.EventID, domain.EventStatusClosed); err != nil {
					return err
				}
			}
			eventErr = domain.ErrEventClosed
			return nil
		}
		if hold.Status == domain.HoldStatusExpired || !hold.ExpiresAt.After(now) {
			return domain.ErrHoldExpired
		}

		order := domain.Order{
			ID:             newUUID(),
			HoldID:         in.HoldID,
			IdempotencyKey: in.IdempotencyKey,
			Status:         domain.OrderStatusConfirmed,
			CreatedAt:      now,
		}

		if err := s.repo.CreateOrder(txCtx, order); err != nil {
			// Re-check for the same idempotency key when a concurrent confirm wins the race.
			if err == domain.ErrHoldAlreadyConfirmed {
				existing, err := s.repo.GetOrderByHoldID(txCtx, in.HoldID)
				if err != nil {
					return err
				}
				if existing != nil && existing.IdempotencyKey == in.IdempotencyKey {
					result = ConfirmHoldResult{Order: *existing, Created: false}
					return nil
				}
			}
			return err
		}
		if err := s.repo.UpdateHoldStatus(txCtx, in.HoldID, domain.HoldStatusConfirmed); err != nil {
			return err
		}

		result = ConfirmHoldResult{Order: order, Created: true}
		return nil
	})
	if err != nil {
		return ConfirmHoldResult{}, err
	}
	if eventErr != nil {
		return ConfirmHoldResult{}, eventErr
	}
	return result, nil
}
