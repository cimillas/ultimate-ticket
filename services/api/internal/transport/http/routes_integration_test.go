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
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	"github.com/cimillas/ultimate-ticket/services/api/internal/testutil"
)

func TestMuxWiring_PublicEvents(t *testing.T) {
	mux, _, _ := newTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var events []eventResponse
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestMuxWiring_HoldsRequireAuth(t *testing.T) {
	mux, _, _ := newTestMux(t)

	req := httptest.NewRequest(http.MethodPost, "/holds", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	var errResp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Code != codeUnauthorized {
		t.Fatalf("expected code %s, got %s", codeUnauthorized, errResp.Code)
	}
}

func TestMuxWiring_AdminRequiresAdmin(t *testing.T) {
	mux, authSvc, cookieCfg := newTestMux(t)

	user, err := authSvc.Register(context.Background(), app.RegisterInput{
		Username: "user",
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	req.AddCookie(&http.Cookie{Name: cookieCfg.Name, Value: user.SessionToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
	var errResp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Code != codeForbidden {
		t.Fatalf("expected code %s, got %s", codeForbidden, errResp.Code)
	}
}

func TestMuxWiring_AdminAllowsAdmin(t *testing.T) {
	mux, authSvc, cookieCfg := newTestMux(t)

	if err := authSvc.BootstrapAdmin(context.Background(), "admin", "admin@example.com", "secret"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	admin, err := authSvc.Login(context.Background(), app.LoginInput{
		Identifier: "admin",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	req.AddCookie(&http.Cookie{Name: cookieCfg.Name, Value: admin.SessionToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func newTestMux(t *testing.T) (http.Handler, *app.AuthService, SessionCookieConfig) {
	t.Helper()

	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)
	testutil.TruncateAll(t, context.Background(), pool)

	holdRepo := postgres.NewHoldRepository(pool)
	holdSvc := app.NewHoldService(holdRepo, clock.NewFixed(time.Now()))
	orderRepo := postgres.NewOrderRepository(pool)
	orderSvc := app.NewOrderService(orderRepo, clock.NewFixed(time.Now()))
	adminRepo := postgres.NewAdminRepository(pool)
	adminSvc := app.NewAdminService(adminRepo, clock.NewFixed(time.Now()))
	authRepo := postgres.NewAuthRepository(pool)
	authSvc := app.NewAuthService(authRepo, clock.NewFixed(time.Now()), time.Hour)

	cookieCfg := SessionCookieConfig{
		Name:     "ut_session",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	mux := http.NewServeMux()
	mux.Handle("/events", HandleEvents(adminSvc))
	mux.Handle("/events/", HandleEventZones(adminSvc))
	mux.Handle("/holds", RequireAuth(authSvc, cookieCfg, HandleCreateHold(holdSvc)))
	mux.Handle("/holds/", RequireAuth(authSvc, cookieCfg, HandleConfirmHold(orderSvc)))
	mux.Handle("/admin/events", RequireAuth(authSvc, cookieCfg, RequireAdmin(HandleAdminEvents(adminSvc))))
	mux.Handle("/admin/events/", RequireAuth(authSvc, cookieCfg, RequireAdmin(HandleAdminZones(adminSvc))))

	return mux, authSvc, cookieCfg
}
