package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestHandleConfirmHold(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 3, 12, 0, 0, 0, time.UTC)
	order := domain.Order{
		ID:             "order-1",
		HoldID:         "hold-1",
		IdempotencyKey: "idem-1",
		CreatedAt:      now,
	}

	tests := []struct {
		name           string
		path           string
		idempotencyKey string
		result         app.ConfirmHoldResult
		serviceErr     error
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "created",
			path:           "/holds/hold-1/confirm",
			idempotencyKey: "idem-1",
			result:         app.ConfirmHoldResult{Order: order, Created: true},
			expectedStatus: http.StatusCreated,
			expectedSubstr: `"hold_id":"hold-1"`,
		},
		{
			name:           "idempotent",
			path:           "/holds/hold-1/confirm",
			idempotencyKey: "idem-1",
			result:         app.ConfirmHoldResult{Order: order, Created: false},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing idempotency header",
			path:           "/holds/hold-1/confirm",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "hold not found",
			path:           "/holds/hold-1/confirm",
			idempotencyKey: "idem-1",
			serviceErr:     domain.ErrHoldNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid id",
			path:           "/holds/not-a-uuid/confirm",
			idempotencyKey: "idem-1",
			serviceErr:     domain.ErrInvalidID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "hold expired",
			path:           "/holds/hold-1/confirm",
			idempotencyKey: "idem-1",
			serviceErr:     domain.ErrHoldExpired,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid path",
			path:           "/holds/hold-1",
			idempotencyKey: "idem-1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &stubHoldConfirmer{
				result: tt.result,
				err:    tt.serviceErr,
			}

			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.idempotencyKey != "" {
				req.Header.Set(idempotencyHeader, tt.idempotencyKey)
			}
			rec := httptest.NewRecorder()

			HandleConfirmHold(svc).ServeHTTP(rec, req)

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

type stubHoldConfirmer struct {
	result app.ConfirmHoldResult
	err    error
}

func (s *stubHoldConfirmer) ConfirmHold(_ context.Context, _ app.ConfirmHoldInput) (app.ConfirmHoldResult, error) {
	return s.result, s.err
}
