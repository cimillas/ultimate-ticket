package app

import (
	"context"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type AdminRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	CreateEvent(ctx context.Context, event domain.Event) error
	ListEvents(ctx context.Context) ([]domain.Event, error)
	CancelEvent(ctx context.Context, eventID string, now time.Time) (domain.Event, error)
	GetEventForUpdate(ctx context.Context, eventID string) (domain.Event, error)
	UpdateEventStatus(ctx context.Context, eventID string, status domain.EventStatus) error
	CreateZone(ctx context.Context, zone domain.Zone) error
	ListZonesByEvent(ctx context.Context, eventID string, now time.Time) ([]domain.Zone, error)
	ListActiveHoldsByZone(ctx context.Context, eventID, zoneID string, now time.Time) ([]domain.Hold, error)
	ListOrdersByZone(ctx context.Context, eventID, zoneID string) ([]domain.Order, error)
}

type AdminService struct {
	repo  AdminRepository
	clock clock.Clock
}

func NewAdminService(repo AdminRepository, clk clock.Clock) *AdminService {
	return &AdminService{
		repo:  repo,
		clock: clk,
	}
}

type CreateEventInput struct {
	Name     string
	StartsAt *time.Time
}

func (s *AdminService) CreateEvent(ctx context.Context, in CreateEventInput) (domain.Event, error) {
	if in.Name == "" {
		return domain.Event{}, domain.ErrEventNameRequired
	}
	startsAt := s.clock.Now()
	if in.StartsAt != nil {
		startsAt = *in.StartsAt
	}

	event := domain.Event{
		ID:       newUUID(),
		Name:     in.Name,
		StartsAt: startsAt,
		Status:   domain.EventStatusActive,
	}

	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return domain.Event{}, err
	}
	return event, nil
}

func (s *AdminService) ListEvents(ctx context.Context) ([]domain.Event, error) {
	events, err := s.repo.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	for i := range events {
		if events[i].Status == "" {
			events[i].Status = domain.EventStatusActive
		}
		zones, err := s.repo.ListZonesByEvent(ctx, events[i].ID, now)
		if err != nil {
			return nil, err
		}
		if len(zones) == 0 {
			events[i].IsComplete = false
			continue
		}
		complete := true
		for _, zone := range zones {
			if zone.Available > 0 {
				complete = false
				break
			}
		}
		events[i].IsComplete = complete
	}

	return events, nil
}

func (s *AdminService) CancelEvent(ctx context.Context, eventID string) (domain.Event, error) {
	if eventID == "" {
		return domain.Event{}, domain.ErrInvalidID
	}
	now := s.clock.Now()
	event, err := s.repo.CancelEvent(ctx, eventID, now)
	if err != nil {
		return domain.Event{}, err
	}

	zones, err := s.repo.ListZonesByEvent(ctx, event.ID, now)
	if err != nil {
		return domain.Event{}, err
	}
	if len(zones) == 0 {
		event.IsComplete = false
		return event, nil
	}
	complete := true
	for _, zone := range zones {
		if zone.Available > 0 {
			complete = false
			break
		}
	}
	event.IsComplete = complete
	return event, nil
}

type CreateZoneInput struct {
	EventID  string
	Name     string
	Capacity int
}

func (s *AdminService) CreateZone(ctx context.Context, in CreateZoneInput) (domain.Zone, error) {
	if in.EventID == "" {
		return domain.Zone{}, domain.ErrInvalidID
	}
	if in.Name == "" {
		return domain.Zone{}, domain.ErrZoneNameRequired
	}
	if in.Capacity <= 0 {
		return domain.Zone{}, domain.ErrInvalidCapacity
	}

	zone := domain.Zone{
		ID:        newUUID(),
		EventID:   in.EventID,
		Name:      in.Name,
		Capacity:  in.Capacity,
		Available: in.Capacity,
	}

	now := s.clock.Now()
	var eventErr error
	if err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		event, err := s.repo.GetEventForUpdate(txCtx, in.EventID)
		if err != nil {
			return err
		}
		if event.Status == domain.EventStatusCancelled {
			eventErr = domain.ErrEventCancelled
			return nil
		}
		if event.Status == domain.EventStatusClosed {
			eventErr = domain.ErrEventClosed
			return nil
		}
		if !event.StartsAt.After(now) {
			if event.Status == "" || event.Status == domain.EventStatusActive {
				if err := s.repo.UpdateEventStatus(txCtx, event.ID, domain.EventStatusClosed); err != nil {
					return err
				}
			}
			eventErr = domain.ErrEventClosed
			return nil
		}
		if err := s.repo.CreateZone(txCtx, zone); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return domain.Zone{}, err
	}
	if eventErr != nil {
		return domain.Zone{}, eventErr
	}
	return zone, nil
}

func (s *AdminService) ListZones(ctx context.Context, eventID string) ([]domain.Zone, error) {
	if eventID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListZonesByEvent(ctx, eventID, s.clock.Now())
}

func (s *AdminService) ListActiveHolds(ctx context.Context, eventID, zoneID string) ([]domain.Hold, error) {
	if eventID == "" || zoneID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListActiveHoldsByZone(ctx, eventID, zoneID, s.clock.Now())
}

func (s *AdminService) ListOrders(ctx context.Context, eventID, zoneID string) ([]domain.Order, error) {
	if eventID == "" || zoneID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListOrdersByZone(ctx, eventID, zoneID)
}
