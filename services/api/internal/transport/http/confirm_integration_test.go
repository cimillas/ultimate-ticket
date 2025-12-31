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
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestConfirmHold_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	repo := postgres.NewOrderRepository(pool)
	svc := app.NewOrderService(repo, clock.NewSystem())

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		Status:         domain.HoldStatusActive,
		Quantity:       2,
		ExpiresAt:      time.Now().UTC().Add(10 * time.Minute),
		IdempotencyKey: "idem-hold",
	})

	handler := HandleConfirmHold(svc)

	req := httptest.NewRequest(http.MethodPost, "/holds/"+holdID+"/confirm", nil)
	req.Header.Set(idempotencyHeader, "idem-confirm")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var first confirmHoldResponse
	if err := json.NewDecoder(rec.Body).Decode(&first); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if first.HoldID != holdID {
		t.Fatalf("expected hold_id %s, got %s", holdID, first.HoldID)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/holds/"+holdID+"/confirm", nil)
	req2.Header.Set(idempotencyHeader, "idem-confirm")
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec2.Code)
	}

	var second confirmHoldResponse
	if err := json.NewDecoder(rec2.Body).Decode(&second); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same order id on idempotent retry")
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM holds WHERE id = $1`, holdID).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != string(domain.HoldStatusConfirmed) {
		t.Fatalf("expected hold status confirmed, got %s", status)
	}
}
