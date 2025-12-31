package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestHandleCreateHold(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	successHold := domain.Hold{
		ID:        "hold-123",
		Status:    domain.HoldStatusActive,
		ExpiresAt: now.Add(15 * time.Minute),
	}

	tests := []struct {
		name           string
		body           string
		serviceErr     error
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "success",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":2,"idempotency_key":"k1"}`,
			expectedStatus: http.StatusCreated,
			expectedSubstr: `"id":"hold-123"`,
		},
		{
			name:           "invalid json",
			body:           `{"event_id":`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing idempotency",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":2}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid quantity",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":0,"idempotency_key":"k1"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zone not found",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			serviceErr:     domain.ErrZoneNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid id",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			serviceErr:     domain.ErrInvalidID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "idempotency conflict",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			serviceErr:     domain.ErrIdempotencyConflict,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "insufficient capacity",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			serviceErr:     domain.ErrInsufficientCapacity,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "internal error",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			serviceErr:     errors.New("boom"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &stubHoldService{
				hold: successHold,
				err:  tt.serviceErr,
			}
			req := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler := HandleCreateHold(svc)
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

type stubHoldService struct {
	hold domain.Hold
	err  error
}

func (s *stubHoldService) CreateHold(_ context.Context, _ app.CreateHoldInput) (domain.Hold, error) {
	return s.hold, s.err
}
