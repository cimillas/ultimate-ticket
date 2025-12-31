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

func TestCreateHold_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := postgres.NewHoldRepository(pool)
	now := time.Date(2025, 1, 4, 10, 0, 0, 0, time.UTC)
	svc := app.NewHoldService(repo, clock.NewFixed(now))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

	body := []byte(`{"event_id":"` + eventID + `","zone_id":"` + zoneID + `","quantity":3,"idempotency_key":"idem-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	HandleCreateHold(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var resp createHoldResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != string(domain.HoldStatusActive) {
		t.Fatalf("expected status active, got %s", resp.Status)
	}
	if resp.ExpiresAt != now.Add(15*time.Minute) {
		t.Fatalf("expected expires_at %v, got %v", now.Add(15*time.Minute), resp.ExpiresAt)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM holds WHERE event_id = $1 AND zone_id = $2 AND idempotency_key = $3`,
		eventID, zoneID, "idem-1",
	).Scan(&count); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 hold, got %d", count)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBuffer(body))
	rec2 := httptest.NewRecorder()
	HandleCreateHold(svc).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on idempotent retry, got %d", rec2.Code)
	}

	var resp2 createHoldResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp2.ID != resp.ID {
		t.Fatalf("expected same hold ID on idempotent retry")
	}

	conflictBody := []byte(`{"event_id":"` + eventID + `","zone_id":"` + zoneID + `","quantity":4,"idempotency_key":"idem-1"}`)
	req3 := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBuffer(conflictBody))
	rec3 := httptest.NewRecorder()
	HandleCreateHold(svc).ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusConflict {
		t.Fatalf("expected status 409 on idempotency conflict, got %d", rec3.Code)
	}
}

func TestCreateAndConfirm_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	holdRepo := postgres.NewHoldRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)

	now := time.Date(2025, 1, 4, 12, 0, 0, 0, time.UTC)
	holdSvc := app.NewHoldService(holdRepo, clock.NewFixed(now))
	orderSvc := app.NewOrderService(orderRepo, clock.NewFixed(now.Add(1*time.Minute)))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)

	mux := http.NewServeMux()
	mux.Handle("/holds", HandleCreateHold(holdSvc))
	mux.Handle("/holds/", HandleConfirmHold(orderSvc))

	body := []byte(`{"event_id":"` + eventID + `","zone_id":"` + zoneID + `","quantity":2,"idempotency_key":"idem-create"}`)
	req := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var created createHoldResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected hold id to be set")
	}

	confirmReq := httptest.NewRequest(http.MethodPost, "/holds/"+created.ID+"/confirm", nil)
	confirmReq.Header.Set(idempotencyHeader, "idem-confirm")
	confirmRec := httptest.NewRecorder()
	mux.ServeHTTP(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", confirmRec.Code)
	}

	var confirmed confirmHoldResponse
	if err := json.NewDecoder(confirmRec.Body).Decode(&confirmed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if confirmed.HoldID != created.ID {
		t.Fatalf("expected hold_id %s, got %s", created.ID, confirmed.HoldID)
	}

	confirmReq2 := httptest.NewRequest(http.MethodPost, "/holds/"+created.ID+"/confirm", nil)
	confirmReq2.Header.Set(idempotencyHeader, "idem-confirm")
	confirmRec2 := httptest.NewRecorder()
	mux.ServeHTTP(confirmRec2, confirmReq2)

	if confirmRec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 on idempotent retry, got %d", confirmRec2.Code)
	}

	var confirmed2 confirmHoldResponse
	if err := json.NewDecoder(confirmRec2.Body).Decode(&confirmed2); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if confirmed2.ID != confirmed.ID {
		t.Fatalf("expected same order id on idempotent retry")
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != string(domain.HoldStatusConfirmed) {
		t.Fatalf("expected hold status confirmed, got %s", status)
	}
}
