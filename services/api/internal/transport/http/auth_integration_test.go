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

type authErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func TestAuthRegisterAndMe_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAuthRepository(pool)
	svc := app.NewAuthService(repo, clock.NewFixed(time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)), time.Hour)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	cookieCfg := SessionCookieConfig{
		Name:     "ut_session",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	registerHandler := HandleRegister(svc, cookieCfg, true)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(`{"username":"ana","email":"ana@example.com","password":"secret"}`),
	)
	registerRec := httptest.NewRecorder()
	registerHandler.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", registerRec.Code)
	}

	res := registerRec.Result()
	defer res.Body.Close()

	var registerResp authResponse
	if err := json.NewDecoder(res.Body).Decode(&registerResp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResp.User.Username != "ana" {
		t.Fatalf("expected username ana, got %s", registerResp.User.Username)
	}
	if registerResp.User.Email != "ana@example.com" {
		t.Fatalf("expected email ana@example.com, got %s", registerResp.User.Email)
	}
	if registerResp.User.Role == "" {
		t.Fatalf("expected role to be set")
	}

	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	sessionCookie := cookies[0]
	if sessionCookie.Name != "ut_session" {
		t.Fatalf("expected cookie ut_session, got %s", sessionCookie.Name)
	}
	if sessionCookie.Value == "" {
		t.Fatalf("expected session cookie value")
	}
	if !sessionCookie.HttpOnly {
		t.Fatalf("expected httpOnly cookie")
	}
	if sessionCookie.Path != "/" {
		t.Fatalf("expected cookie path /, got %s", sessionCookie.Path)
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax, got %v", sessionCookie.SameSite)
	}

	meHandler := RequireAuth(svc, cookieCfg, HandleMe())
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	meHandler.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", meRec.Code)
	}

	var meResp authResponse
	if err := json.NewDecoder(meRec.Body).Decode(&meResp); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meResp.User.ID != registerResp.User.ID {
		t.Fatalf("expected same user id, got %s", meResp.User.ID)
	}
}

func TestAuthRegister_InvalidUsername_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAuthRepository(pool)
	svc := app.NewAuthService(repo, clock.NewFixed(time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)), time.Hour)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	registerHandler := HandleRegister(svc, SessionCookieConfig{Name: "ut_session"}, true)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(`{"username":"bad@example.com","email":"ana@example.com","password":"secret"}`),
	)
	registerRec := httptest.NewRecorder()
	registerHandler.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", registerRec.Code)
	}

	var errResp authErrorResponse
	if err := json.NewDecoder(registerRec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Code != codeUsernameInvalid {
		t.Fatalf("expected code %s, got %s", codeUsernameInvalid, errResp.Code)
	}
}

func TestAuthLogin_InvalidCredentials_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAuthRepository(pool)
	svc := app.NewAuthService(repo, clock.NewFixed(time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)), time.Hour)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	loginHandler := HandleLogin(svc, SessionCookieConfig{Name: "ut_session"})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login",
		bytes.NewBufferString(`{"identifier":"missing","password":"wrong"}`),
	)
	loginRec := httptest.NewRecorder()
	loginHandler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", loginRec.Code)
	}

	var errResp apiErrorResponse
	if err := json.NewDecoder(loginRec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Code != codeInvalidCredentials {
		t.Fatalf("expected code %s, got %s", codeInvalidCredentials, errResp.Code)
	}
}

func TestAuthLogout_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAuthRepository(pool)
	svc := app.NewAuthService(repo, clock.NewFixed(time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)), time.Hour)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	cookieCfg := SessionCookieConfig{Name: "ut_session", Path: "/", SameSite: http.SameSiteLaxMode}
	registerHandler := HandleRegister(svc, cookieCfg, true)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(`{"username":"ben","email":"ben@example.com","password":"secret"}`),
	)
	registerRec := httptest.NewRecorder()
	registerHandler.ServeHTTP(registerRec, registerReq)
	sessionCookie := registerRec.Result().Cookies()[0]

	logoutHandler := HandleLogout(svc, cookieCfg)
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()
	logoutHandler.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logoutRec.Code)
	}

	cleared := logoutRec.Result().Cookies()
	if len(cleared) != 1 {
		t.Fatalf("expected cleared cookie, got %d", len(cleared))
	}
	if cleared[0].Value != "" || cleared[0].MaxAge >= 0 {
		t.Fatalf("expected cleared session cookie")
	}
}

func TestAuthRegister_Disabled_HTTPIntegration(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.ApplyMigrations(t, context.Background(), pool)

	repo := postgres.NewAuthRepository(pool)
	svc := app.NewAuthService(repo, clock.NewFixed(time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)), time.Hour)

	ctx := context.Background()
	testutil.TruncateAll(t, ctx, pool)

	registerHandler := HandleRegister(svc, SessionCookieConfig{Name: "ut_session"}, false)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(`{"username":"ana","email":"ana@example.com","password":"secret"}`),
	)
	registerRec := httptest.NewRecorder()
	registerHandler.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", registerRec.Code)
	}

	var errResp authErrorResponse
	if err := json.NewDecoder(registerRec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Code != codeRegistrationDisabled {
		t.Fatalf("expected code %s, got %s", codeRegistrationDisabled, errResp.Code)
	}
}
