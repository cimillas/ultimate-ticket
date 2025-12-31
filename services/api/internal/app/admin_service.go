package app

import (
	"context"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type AdminRepository interface {
	CreateEvent(ctx context.Context, event domain.Event) error
	ListEvents(ctx context.Context) ([]domain.Event, error)
	CreateZone(ctx context.Context, zone domain.Zone) error
	ListZonesByEvent(ctx context.Context, eventID string) ([]domain.Zone, error)
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
	}

	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return domain.Event{}, err
	}
	return event, nil
}

func (s *AdminService) ListEvents(ctx context.Context) ([]domain.Event, error) {
	return s.repo.ListEvents(ctx)
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
		ID:       newUUID(),
		EventID:  in.EventID,
		Name:     in.Name,
		Capacity: in.Capacity,
	}

	if err := s.repo.CreateZone(ctx, zone); err != nil {
		return domain.Zone{}, err
	}
	return zone, nil
}

func (s *AdminService) ListZones(ctx context.Context, eventID string) ([]domain.Zone, error) {
	if eventID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListZonesByEvent(ctx, eventID)
}
