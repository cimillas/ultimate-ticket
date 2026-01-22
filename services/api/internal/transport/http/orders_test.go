package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestHandleListOrders(t *testing.T) {
	t.Parallel()

	orders := []domain.OrderSummary{
		{
			ID:        "order-1",
			HoldID:    "hold-1",
			EventID:   "event-1",
			ZoneID:    "zone-1",
			Quantity:  2,
			Status:    domain.OrderStatusConfirmed,
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name           string
		withAuth       bool
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "success",
			withAuth:       true,
			expectedStatus: http.StatusOK,
			expectedSubstr: `"id":"order-1"`,
		},
		{
			name:           "missing auth",
			withAuth:       false,
			expectedStatus: http.StatusUnauthorized,
			expectedSubstr: `"code":"unauthorized"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &stubOrderListService{
				orders: orders,
			}
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			if tt.withAuth {
				req = withAuth(req, "user-1")
			}
			rec := httptest.NewRecorder()

			handler := HandleOrders(svc)
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			if res.StatusCode != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}
			if tt.expectedSubstr != "" {
				body := rec.Body.String()
				if !strings.Contains(body, tt.expectedSubstr) {
					t.Fatalf("expected response to contain %q, got %q", tt.expectedSubstr, body)
				}
			}
		})
	}
}

type stubOrderListService struct {
	orders []domain.OrderSummary
	err    error
}

func (s *stubOrderListService) ListOrdersByUser(_ context.Context, _ string) ([]domain.OrderSummary, error) {
	return s.orders, s.err
}
