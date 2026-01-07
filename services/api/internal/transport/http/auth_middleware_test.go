package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type fakeAuthService struct {
	session AuthSession
	err     error
}

func (f fakeAuthService) Authenticate(ctx context.Context, token string) (AuthSession, error) {
	return f.session, f.err
}

func TestRequireAuth_UnauthorizedWithoutCookie(t *testing.T) {
	handler := RequireAuth(fakeAuthService{}, SessionCookieConfig{Name: "ut_session"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/holds", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuth_AttachesUserAndRefreshesCookie(t *testing.T) {
	session := AuthSession{
		User: domain.User{
			ID:       "user-1",
			Username: "ana",
			Role:     domain.UserRoleUser,
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	handler := RequireAuth(fakeAuthService{session: session}, SessionCookieConfig{
		Name:     "ut_session",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := AuthFromContext(r.Context())
		if !ok || got.User.ID != "user-1" {
			t.Fatalf("expected auth context to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/holds", nil)
	req.AddCookie(&http.Cookie{Name: "ut_session", Value: "token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if len(rec.Result().Cookies()) != 1 {
		t.Fatalf("expected refreshed cookie")
	}
}

func TestRequireAdmin_RejectsNonAdmin(t *testing.T) {
	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	req = req.WithContext(WithAuth(req.Context(), AuthSession{
		User: domain.User{ID: "user-1", Role: domain.UserRoleUser},
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	req = req.WithContext(WithAuth(req.Context(), AuthSession{
		User: domain.User{ID: "admin-1", Role: domain.UserRoleAdmin},
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
