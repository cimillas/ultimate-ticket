package http

import (
	"bytes"
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

type apiErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func TestAdminEvents_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	handler := HandleAdminEvents(svc)

	reqBody := []byte(`{"name":"Concert","starts_at":"2025-02-01T10:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/events", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var created eventResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected event id to be set")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRec.Code)
	}

	var events []eventResponse
	if err := json.NewDecoder(listRec.Body).Decode(&events); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestAdminZones_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, _ := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

	handler := HandleAdminZones(svc)

	reqBody := []byte(`{"name":"Zone B","capacity":40}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/events/"+eventID+"/zones", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var created zoneResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.EventID != eventID {
		t.Fatalf("expected event id %s, got %s", eventID, created.EventID)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/events/"+eventID+"/zones", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRec.Code)
	}

	var zones []zoneResponse
	if err := json.NewDecoder(listRec.Body).Decode(&zones); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}

	invalidReq := httptest.NewRequest(http.MethodGet, "/admin/events/not-a-uuid/zones", nil)
	invalidRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidRec, invalidReq)

	if invalidRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", invalidRec.Code)
	}

	var errResp apiErrorResponse
	if err := json.NewDecoder(invalidRec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code != codeInvalidID {
		t.Fatalf("expected error code %s, got %s", codeInvalidID, errResp.Code)
	}
}
