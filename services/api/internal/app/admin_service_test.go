package app

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type fakeAdminRepo struct {
	createdEvent domain.Event
	createdZone  domain.Zone

	createEventErr error
	createZoneErr  error
}

func (f *fakeAdminRepo) CreateEvent(ctx context.Context, event domain.Event) error {
	f.createdEvent = event
	return f.createEventErr
}

func (f *fakeAdminRepo) ListEvents(ctx context.Context) ([]domain.Event, error) {
	return nil, nil
}

func (f *fakeAdminRepo) CreateZone(ctx context.Context, zone domain.Zone) error {
	f.createdZone = zone
	return f.createZoneErr
}

func (f *fakeAdminRepo) ListZonesByEvent(ctx context.Context, eventID string) ([]domain.Zone, error) {
	return nil, nil
}

func TestAdminService_CreateEvent_DefaultStartsAt(t *testing.T) {
	repo := &fakeAdminRepo{}
	now := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)
	svc := NewAdminService(repo, clock.NewFixed(now))

	got, err := svc.CreateEvent(context.Background(), CreateEventInput{Name: "Concert"})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if got.Name != "Concert" {
		t.Fatalf("expected name, got %q", got.Name)
	}
	if got.StartsAt != now {
		t.Fatalf("expected starts_at %v, got %v", now, got.StartsAt)
	}
	if repo.createdEvent.ID == "" {
		t.Fatalf("expected event ID to be set")
	}
}

func TestAdminService_CreateEvent_ValidatesName(t *testing.T) {
	repo := &fakeAdminRepo{}
	svc := NewAdminService(repo, clock.NewFixed(time.Now()))

	_, err := svc.CreateEvent(context.Background(), CreateEventInput{Name: ""})
	if err != domain.ErrEventNameRequired {
		t.Fatalf("expected ErrEventNameRequired, got %v", err)
	}
}

func TestAdminService_CreateZone_ValidatesInput(t *testing.T) {
	repo := &fakeAdminRepo{}
	svc := NewAdminService(repo, clock.NewFixed(time.Now()))
	ctx := context.Background()

	_, err := svc.CreateZone(ctx, CreateZoneInput{EventID: "", Name: "Zone A", Capacity: 10})
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.CreateZone(ctx, CreateZoneInput{EventID: "event", Name: "", Capacity: 10})
	if err != domain.ErrZoneNameRequired {
		t.Fatalf("expected ErrZoneNameRequired, got %v", err)
	}

	_, err = svc.CreateZone(ctx, CreateZoneInput{EventID: "event", Name: "Zone A", Capacity: 0})
	if err != domain.ErrInvalidCapacity {
		t.Fatalf("expected ErrInvalidCapacity, got %v", err)
	}
}
