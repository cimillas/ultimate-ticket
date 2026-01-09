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
		withAuth       bool
		serviceErr     error
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "success",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":2,"idempotency_key":"k1"}`,
			withAuth:       true,
			expectedStatus: http.StatusCreated,
			expectedSubstr: `"id":"hold-123"`,
		},
		{
			name:           "missing auth",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":2,"idempotency_key":"k1"}`,
			withAuth:       false,
			expectedStatus: http.StatusUnauthorized,
			expectedSubstr: `"code":"unauthorized"`,
		},
		{
			name:           "invalid json",
			body:           `{"event_id":`,
			withAuth:       true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing idempotency",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":2}`,
			withAuth:       true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid quantity",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":0,"idempotency_key":"k1"}`,
			withAuth:       true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zone not found",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrZoneNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid id",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrInvalidID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "idempotency conflict",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrIdempotencyConflict,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "insufficient capacity",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrInsufficientCapacity,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "event closed",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrEventClosed,
			expectedStatus: http.StatusConflict,
			expectedSubstr: `"code":"event_closed"`,
		},
		{
			name:           "event cancelled",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
			serviceErr:     domain.ErrEventCancelled,
			expectedStatus: http.StatusConflict,
			expectedSubstr: `"code":"event_cancelled"`,
		},
		{
			name:           "internal error",
			body:           `{"event_id":"e1","zone_id":"z1","quantity":1,"idempotency_key":"k1"}`,
			withAuth:       true,
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
			if tt.withAuth {
				req = withAuth(req, "user-1")
			}
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

func withAuth(req *http.Request, userID string) *http.Request {
	session := AuthSession{
		User: domain.User{
			ID:   userID,
			Role: domain.UserRoleUser,
		},
	}
	return req.WithContext(WithAuth(req.Context(), session))
}

type stubHoldService struct {
	hold domain.Hold
	err  error
}

func (s *stubHoldService) CreateHold(_ context.Context, _ app.CreateHoldInput) (domain.Hold, error) {
	return s.hold, s.err
}
