package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestHandleChangePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           string
		withAuth       bool
		withCookie     bool
		serviceErr     error
		expectedStatus int
		expectedSubstr string
	}{
		{
			name:           "success",
			body:           `{"current_password":"old","new_password":"new"}`,
			withAuth:       true,
			withCookie:     true,
			expectedStatus: http.StatusOK,
			expectedSubstr: `"ok":true`,
		},
		{
			name:           "missing auth",
			body:           `{"current_password":"old","new_password":"new"}`,
			withAuth:       false,
			expectedStatus: http.StatusUnauthorized,
			expectedSubstr: `"code":"unauthorized"`,
		},
		{
			name:           "invalid json",
			body:           `{"current_password":`,
			withAuth:       true,
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid credentials",
			body:           `{"current_password":"old","new_password":"new"}`,
			withAuth:       true,
			withCookie:     true,
			serviceErr:     domain.ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
			expectedSubstr: `"code":"invalid_credentials"`,
		},
		{
			name:           "missing cookie",
			body:           `{"current_password":"old","new_password":"new"}`,
			withAuth:       true,
			withCookie:     false,
			expectedStatus: http.StatusUnauthorized,
			expectedSubstr: `"code":"unauthorized"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &stubPasswordService{err: tt.serviceErr}
			req := httptest.NewRequest(http.MethodPost, "/auth/password", bytes.NewBufferString(tt.body))
			if tt.withAuth {
				req = withAuth(req, "user-1")
			}
			if tt.withCookie {
				req.AddCookie(&http.Cookie{Name: "ut_session", Value: "token"})
			}
			rec := httptest.NewRecorder()

			handler := HandleChangePassword(svc, SessionCookieConfig{Name: "ut_session"})
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

type stubPasswordService struct {
	err error
}

func (s *stubPasswordService) Register(_ context.Context, _ app.RegisterInput) (app.AuthResult, error) {
	return app.AuthResult{}, nil
}

func (s *stubPasswordService) Login(_ context.Context, _ app.LoginInput) (app.AuthResult, error) {
	return app.AuthResult{}, nil
}

func (s *stubPasswordService) Authenticate(_ context.Context, _ string) (app.AuthSession, error) {
	return app.AuthSession{}, nil
}

func (s *stubPasswordService) Logout(_ context.Context, _ string) error {
	return nil
}

func (s *stubPasswordService) ChangePassword(_ context.Context, _ app.ChangePasswordInput) error {
	return s.err
}
