package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestPublicEvents_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, _ := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 50)

	handler := HandleEvents(svc)
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var events []eventResponse
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != eventID {
		t.Fatalf("expected event id %s, got %s", eventID, events[0].ID)
	}
}

func TestPublicEventZones_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 80)

	handler := HandleEventZones(svc)
	req := httptest.NewRequest(http.MethodGet, "/events/"+eventID+"/zones", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var zones []zoneResponse
	if err := json.NewDecoder(rec.Body).Decode(&zones); err != nil {
		t.Fatalf("decode zones: %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if zones[0].ID != zoneID {
		t.Fatalf("expected zone id %s, got %s", zoneID, zones[0].ID)
	}
}
