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

func TestListOrders_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	now := time.Date(2025, 1, 4, 11, 0, 0, 0, time.UTC)
	repo := postgres.NewOrderRepository(pool)
	svc := app.NewOrderService(repo, clock.NewFixed(now))

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)
	eventID, zoneID := testutil.InsertEventAndZone(t, ctx, pool, "Concert", 100)
	userID := testutil.InsertUser(t, ctx, pool, domain.UserRoleUser)
	otherUserID := testutil.InsertUser(t, ctx, pool, domain.UserRoleUser)

	holdID := testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		UserID:         userID,
		Status:         domain.HoldStatusConfirmed,
		Quantity:       2,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "idem-hold",
	})
	testutil.InsertHold(t, ctx, pool, eventID, zoneID, domain.Hold{
		UserID:         otherUserID,
		Status:         domain.HoldStatusConfirmed,
		Quantity:       1,
		ExpiresAt:      now.Add(10 * time.Minute),
		IdempotencyKey: "idem-other",
	})

	if _, err := pool.Exec(ctx, `
INSERT INTO orders (id, hold_id, idempotency_key, status, created_at)
VALUES ($1, $2, $3, $4, $5)`,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		holdID,
		"idem-order",
		domain.OrderStatusConfirmed,
		now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req = withAuth(req, userID)
	rec := httptest.NewRecorder()

	HandleOrders(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp []orderSummaryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 order, got %d", len(resp))
	}
	if resp[0].HoldID != holdID || resp[0].EventID != eventID || resp[0].ZoneID != zoneID {
		t.Fatalf("unexpected order response: %+v", resp[0])
	}
}
