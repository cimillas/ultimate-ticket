package app

import (
	"context"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type OrderRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	GetHoldForUpdate(ctx context.Context, holdID string) (domain.Hold, error)
	GetOrderByHoldID(ctx context.Context, holdID string) (*domain.Order, error)
	CreateOrder(ctx context.Context, order domain.Order) error
	UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error
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

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		hold, err := s.repo.GetHoldForUpdate(txCtx, in.HoldID)
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

		if hold.Status == domain.HoldStatusConfirmed {
			return domain.ErrHoldAlreadyConfirmed
		}
		if hold.Status == domain.HoldStatusExpired || !hold.ExpiresAt.After(now) {
			return domain.ErrHoldExpired
		}

		order := domain.Order{
			ID:             newUUID(),
			HoldID:         in.HoldID,
			IdempotencyKey: in.IdempotencyKey,
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
	return result, nil
}
