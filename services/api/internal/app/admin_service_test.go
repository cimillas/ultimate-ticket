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
	listEventsErr  error
	listZonesErr   error
	listHoldsErr   error
	listOrdersErr  error
	cancelEventErr error

	events         []domain.Event
	eventsByID     map[string]domain.Event
	zonesByEvent   map[string][]domain.Zone
	holdsByZone    map[string][]domain.Hold
	ordersByZone   map[string][]domain.Order
	lastHoldsNow   time.Time
	cancelEventNow time.Time
	cancelEventID  string
	cancelResult   domain.Event
	updatedEvents  map[string]domain.EventStatus
}

func (f *fakeAdminRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeAdminRepo) CreateEvent(ctx context.Context, event domain.Event) error {
	f.createdEvent = event
	return f.createEventErr
}

func (f *fakeAdminRepo) ListEvents(ctx context.Context) ([]domain.Event, error) {
	if f.listEventsErr != nil {
		return nil, f.listEventsErr
	}
	return append([]domain.Event{}, f.events...), nil
}

func (f *fakeAdminRepo) CreateZone(ctx context.Context, zone domain.Zone) error {
	f.createdZone = zone
	return f.createZoneErr
}

func (f *fakeAdminRepo) ListZonesByEvent(ctx context.Context, eventID string, _ time.Time) ([]domain.Zone, error) {
	if f.listZonesErr != nil {
		return nil, f.listZonesErr
	}
	return append([]domain.Zone{}, f.zonesByEvent[eventID]...), nil
}

func (f *fakeAdminRepo) ListActiveHoldsByZone(ctx context.Context, eventID, zoneID string, now time.Time) ([]domain.Hold, error) {
	f.lastHoldsNow = now
	if f.listHoldsErr != nil {
		return nil, f.listHoldsErr
	}
	return append([]domain.Hold{}, f.holdsByZone[eventID+"|"+zoneID]...), nil
}

func (f *fakeAdminRepo) ListOrdersByZone(ctx context.Context, eventID, zoneID string) ([]domain.Order, error) {
	if f.listOrdersErr != nil {
		return nil, f.listOrdersErr
	}
	return append([]domain.Order{}, f.ordersByZone[eventID+"|"+zoneID]...), nil
}

func (f *fakeAdminRepo) CancelEvent(ctx context.Context, eventID string, now time.Time) (domain.Event, error) {
	f.cancelEventID = eventID
	f.cancelEventNow = now
	return f.cancelResult, f.cancelEventErr
}

func (f *fakeAdminRepo) GetEventForUpdate(ctx context.Context, eventID string) (domain.Event, error) {
	if f.eventsByID == nil {
		return domain.Event{}, domain.ErrEventNotFound
	}
	event, ok := f.eventsByID[eventID]
	if !ok {
		return domain.Event{}, domain.ErrEventNotFound
	}
	return event, nil
}

func (f *fakeAdminRepo) UpdateEventStatus(ctx context.Context, eventID string, status domain.EventStatus) error {
	if f.updatedEvents == nil {
		f.updatedEvents = make(map[string]domain.EventStatus)
	}
	f.updatedEvents[eventID] = status
	if f.eventsByID == nil {
		f.eventsByID = make(map[string]domain.Event)
	}
	event := f.eventsByID[eventID]
	event.ID = eventID
	event.Status = status
	f.eventsByID[eventID] = event
	return nil
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

func TestAdminService_CreateZone_ClosesEventWhenStarted(t *testing.T) {
	now := time.Date(2025, 2, 2, 10, 0, 0, 0, time.UTC)
	repo := &fakeAdminRepo{
		eventsByID: map[string]domain.Event{
			"event-1": {
				ID:       "event-1",
				StartsAt: now.Add(-1 * time.Minute),
				Status:   domain.EventStatusActive,
			},
		},
	}
	svc := NewAdminService(repo, clock.NewFixed(now))

	_, err := svc.CreateZone(context.Background(), CreateZoneInput{
		EventID:  "event-1",
		Name:     "Zone A",
		Capacity: 10,
	})
	if err != domain.ErrEventClosed {
		t.Fatalf("expected ErrEventClosed, got %v", err)
	}
	if repo.createdZone.ID != "" {
		t.Fatalf("expected zone not to be created")
	}
	if repo.updatedEvents["event-1"] != domain.EventStatusClosed {
		t.Fatalf("expected event to be closed, got %s", repo.updatedEvents["event-1"])
	}
}

func TestAdminService_ListEvents_CompletesWhenAllZonesUnavailable(t *testing.T) {
	repo := &fakeAdminRepo{
		events: []domain.Event{
			{ID: "event-1", Name: "Concert", StartsAt: time.Now()},
			{ID: "event-2", Name: "Festival", StartsAt: time.Now()},
			{ID: "event-3", Name: "No Zones", StartsAt: time.Now()},
		},
		zonesByEvent: map[string][]domain.Zone{
			"event-1": {
				{ID: "zone-1", EventID: "event-1", Capacity: 10, Available: 0},
				{ID: "zone-2", EventID: "event-1", Capacity: 5, Available: 0},
			},
			"event-2": {
				{ID: "zone-3", EventID: "event-2", Capacity: 10, Available: 3},
			},
		},
	}

	svc := NewAdminService(repo, clock.NewFixed(time.Now()))

	events, err := svc.ListEvents(context.Background())
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if !events[0].IsComplete {
		t.Fatalf("expected event-1 to be complete")
	}
	if events[1].IsComplete {
		t.Fatalf("expected event-2 to be incomplete")
	}
	if events[2].IsComplete {
		t.Fatalf("expected event-3 to be incomplete with no zones")
	}
}

func TestAdminService_ListActiveHolds_ValidatesInput(t *testing.T) {
	repo := &fakeAdminRepo{}
	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)
	svc := NewAdminService(repo, clock.NewFixed(now))
	ctx := context.Background()

	_, err := svc.ListActiveHolds(ctx, "", "zone-1")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	_, err = svc.ListActiveHolds(ctx, "event-1", "")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestAdminService_ListActiveHolds_Delegates(t *testing.T) {
	repo := &fakeAdminRepo{
		holdsByZone: map[string][]domain.Hold{
			"event-1|zone-1": {{ID: "hold-1"}},
		},
	}
	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)
	svc := NewAdminService(repo, clock.NewFixed(now))

	holds, err := svc.ListActiveHolds(context.Background(), "event-1", "zone-1")
	if err != nil {
		t.Fatalf("list holds: %v", err)
	}
	if len(holds) != 1 || holds[0].ID != "hold-1" {
		t.Fatalf("unexpected holds: %+v", holds)
	}
	if !repo.lastHoldsNow.Equal(now) {
		t.Fatalf("expected now %v, got %v", now, repo.lastHoldsNow)
	}
}

func TestAdminService_ListOrders_ValidatesInput(t *testing.T) {
	repo := &fakeAdminRepo{}
	svc := NewAdminService(repo, clock.NewFixed(time.Now()))
	ctx := context.Background()

	_, err := svc.ListOrders(ctx, "", "zone-1")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	_, err = svc.ListOrders(ctx, "event-1", "")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestAdminService_ListOrders_Delegates(t *testing.T) {
	repo := &fakeAdminRepo{
		ordersByZone: map[string][]domain.Order{
			"event-1|zone-1": {{ID: "order-1"}},
		},
	}
	svc := NewAdminService(repo, clock.NewFixed(time.Now()))

	orders, err := svc.ListOrders(context.Background(), "event-1", "zone-1")
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != "order-1" {
		t.Fatalf("unexpected orders: %+v", orders)
	}
}

func TestAdminService_CancelEvent_ValidatesInput(t *testing.T) {
	repo := &fakeAdminRepo{}
	svc := NewAdminService(repo, clock.NewFixed(time.Now()))

	_, err := svc.CancelEvent(context.Background(), "")
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestAdminService_CancelEvent_Delegates(t *testing.T) {
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	cancelledAt := now
	repo := &fakeAdminRepo{
		cancelResult: domain.Event{
			ID:          "event-1",
			Status:      domain.EventStatusCancelled,
			CancelledAt: &cancelledAt,
		},
	}
	svc := NewAdminService(repo, clock.NewFixed(now))

	evt, err := svc.CancelEvent(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("cancel event: %v", err)
	}
	if repo.cancelEventID != "event-1" {
		t.Fatalf("expected event id to be passed, got %q", repo.cancelEventID)
	}
	if !repo.cancelEventNow.Equal(now) {
		t.Fatalf("expected cancel time %v, got %v", now, repo.cancelEventNow)
	}
	if evt.Status != domain.EventStatusCancelled || evt.CancelledAt == nil {
		t.Fatalf("expected cancelled event, got %+v", evt)
	}
}

func TestAdminService_CancelEvent_SetsIsComplete(t *testing.T) {
	now := time.Date(2025, 2, 2, 12, 0, 0, 0, time.UTC)
	repo := &fakeAdminRepo{
		cancelResult: domain.Event{
			ID:     "event-2",
			Status: domain.EventStatusCancelled,
		},
		zonesByEvent: map[string][]domain.Zone{
			"event-2": {
				{ID: "zone-1", EventID: "event-2", Capacity: 10, Available: 0},
				{ID: "zone-2", EventID: "event-2", Capacity: 5, Available: 0},
			},
		},
	}
	svc := NewAdminService(repo, clock.NewFixed(now))

	evt, err := svc.CancelEvent(context.Background(), "event-2")
	if err != nil {
		t.Fatalf("cancel event: %v", err)
	}
	if !evt.IsComplete {
		t.Fatalf("expected event to be complete, got %+v", evt)
	}
}
