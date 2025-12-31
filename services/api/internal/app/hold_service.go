package app

import (
	"context"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type HoldRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	GetZoneForUpdate(ctx context.Context, eventID, zoneID string) (domain.Zone, error)
	FindHoldByIdempotencyKey(ctx context.Context, eventID, zoneID, key string) (*domain.Hold, error)
	SumActiveHolds(ctx context.Context, eventID, zoneID string, now time.Time) (int, error)
	SumConfirmed(ctx context.Context, eventID, zoneID string) (int, error)
	CreateHold(ctx context.Context, hold domain.Hold) error
}

type HoldService struct {
	repo    HoldRepository
	clock   clock.Clock
	holdTTL time.Duration
}

const defaultHoldTTL = 15 * time.Minute

func NewHoldService(repo HoldRepository, clk clock.Clock, opts ...HoldServiceOption) *HoldService {
	svc := &HoldService{
		repo:    repo,
		clock:   clk,
		holdTTL: defaultHoldTTL,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

type HoldServiceOption func(*HoldService)

// WithHoldTTL overrides the default TTL for new holds.
func WithHoldTTL(d time.Duration) HoldServiceOption {
	return func(s *HoldService) {
		if d > 0 {
			s.holdTTL = d
		}
	}
}

type CreateHoldInput struct {
	EventID        string
	ZoneID         string
	Quantity       int
	IdempotencyKey string
}

func (s *HoldService) CreateHold(ctx context.Context, in CreateHoldInput) (domain.Hold, error) {
	if in.Quantity <= 0 {
		return domain.Hold{}, domain.ErrInvalidQuantity
	}
	if in.IdempotencyKey == "" {
		return domain.Hold{}, domain.ErrIdempotencyKeyRequired
	}

	now := s.clock.Now()
	var result domain.Hold

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if existing, err := s.repo.FindHoldByIdempotencyKey(txCtx, in.EventID, in.ZoneID, in.IdempotencyKey); err != nil {
			return err
		} else if existing != nil {
			if existing.Quantity != in.Quantity {
				return domain.ErrIdempotencyConflict
			}
			result = *existing
			return nil
		}

		zone, err := s.repo.GetZoneForUpdate(txCtx, in.EventID, in.ZoneID)
		if err != nil {
			return err
		}

		activeQty, err := s.repo.SumActiveHolds(txCtx, in.EventID, in.ZoneID, now)
		if err != nil {
			return err
		}
		confirmedQty, err := s.repo.SumConfirmed(txCtx, in.EventID, in.ZoneID)
		if err != nil {
			return err
		}

		available := zone.Capacity - activeQty - confirmedQty
		if in.Quantity > available {
			return domain.ErrInsufficientCapacity
		}

		hold := domain.Hold{
			ID:             newUUID(),
			EventID:        in.EventID,
			ZoneID:         in.ZoneID,
			Quantity:       in.Quantity,
			Status:         domain.HoldStatusActive,
			ExpiresAt:      now.Add(s.holdTTL),
			IdempotencyKey: in.IdempotencyKey,
			CreatedAt:      now,
		}

		if err := s.repo.CreateHold(txCtx, hold); err != nil {
			// Re-read on conflict to keep idempotent retries consistent under concurrency.
			if err == domain.ErrIdempotencyConflict {
				existing, err := s.repo.FindHoldByIdempotencyKey(txCtx, in.EventID, in.ZoneID, in.IdempotencyKey)
				if err != nil {
					return err
				}
				if existing != nil {
					if existing.Quantity != in.Quantity {
						return domain.ErrIdempotencyConflict
					}
					result = *existing
					return nil
				}
			}
			return err
		}

		result = hold
		return nil
	})
	if err != nil {
		return domain.Hold{}, err
	}

	return result, nil
}
