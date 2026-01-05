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
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
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
	if events[0].IsComplete {
		t.Fatalf("expected event to be incomplete with no zones")
	}
	if events[0].Status != "active" {
		t.Fatalf("expected status active, got %s", events[0].Status)
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
	if created.Available != created.Capacity {
		t.Fatalf("expected available %d, got %d", created.Capacity, created.Available)
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
	for _, zone := range zones {
		if zone.Available != zone.Capacity {
			t.Fatalf("expected available %d for zone %s, got %d", zone.Capacity, zone.ID, zone.Available)
		}
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

func TestAdminZoneHolds_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	now := time.Date(2025, 1, 6, 11, 0, 0, 0, time.UTC)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(now))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	activeHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "active-1",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       1,
		ExpiresAt:      now.Add(-1 * time.Minute),
		IdempotencyKey: "expired-1",
	})

	handler := HandleAdminZones(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/events/"+eventID+"/zones/"+zoneID+"/holds", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var holds []adminHoldResponse
	if err := json.NewDecoder(rec.Body).Decode(&holds); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(holds) != 1 {
		t.Fatalf("expected 1 hold, got %d", len(holds))
	}
	if holds[0].ID != activeHoldID {
		t.Fatalf("expected hold %s, got %s", activeHoldID, holds[0].ID)
	}
}

func TestAdminZoneOrders_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	now := time.Date(2025, 1, 6, 11, 0, 0, 0, time.UTC)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(now))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "hold-1",
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, created_at)
VALUES ($1, $2, $3, $4)`,
		"00000000-0000-0000-0000-000000000090", holdID, "order-1", now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	handler := HandleAdminZones(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/events/"+eventID+"/zones/"+zoneID+"/orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var orders []adminOrderResponse
	if err := json.NewDecoder(rec.Body).Decode(&orders); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].HoldID != holdID {
		t.Fatalf("expected hold_id %s, got %s", holdID, orders[0].HoldID)
	}
}

func TestAdminEvents_Cancel_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	now := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)

	repo := postgres.NewAdminRepository(pool)
	svc := app.NewAdminService(repo, clock.NewFixed(now))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	activeHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "active-1",
	})
	confirmedHoldID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "confirmed-1",
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, created_at)
VALUES ($1, $2, $3, $4)`,
		"00000000-0000-0000-0000-000000000099", confirmedHoldID, "order-1", now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	handler := HandleAdminZones(svc)
	cancelReq := httptest.NewRequest(http.MethodPost, "/admin/events/"+eventID+"/cancel", nil)
	cancelRec := httptest.NewRecorder()
	handler.ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", cancelRec.Code)
	}

	var cancelled eventResponse
	if err := json.NewDecoder(cancelRec.Body).Decode(&cancelled); err != nil {
		t.Fatalf("decode cancel response: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("expected status cancelled, got %s", cancelled.Status)
	}
	if cancelled.CancelledAt == nil {
		t.Fatalf("expected cancelled_at to be set")
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, activeHoldID).Scan(&status); err != nil {
		t.Fatalf("query active hold status: %v", err)
	}
	if status != string(domain.HoldStatusInvalid) {
		t.Fatalf("expected active hold invalid, got %s", status)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, confirmedHoldID).Scan(&status); err != nil {
		t.Fatalf("query confirmed hold status: %v", err)
	}
	if status != string(domain.HoldStatusInvalid) {
		t.Fatalf("expected confirmed hold invalid, got %s", status)
	}

	var orderStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE hold_id = $1`, confirmedHoldID).Scan(&orderStatus); err != nil {
		t.Fatalf("query order status: %v", err)
	}
	if orderStatus != string(domain.OrderStatusRefunded) {
		t.Fatalf("expected order refunded, got %s", orderStatus)
	}
}
