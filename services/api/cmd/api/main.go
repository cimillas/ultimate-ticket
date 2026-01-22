package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/config"
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	transporthttp "github.com/cimillas/ultimate-ticket/services/api/internal/transport/http"
	"github.com/cimillas/ultimate-ticket/services/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDatabaseURL = "postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable"
const defaultPort = "8080"
const defaultCORSOrigins = "http://localhost:5173,http://127.0.0.1:5173"
const defaultSessionTTL = time.Hour
const defaultSessionCookieName = "ut_session"
const shutdownTimeout = 10 * time.Second

func main() {
	logger := log.Default()
	config.LoadEnvFile(logger)

	port := os.Getenv("PORT")
	if port == "" {
		logger.Printf("WARN: PORT not set, using default %s", defaultPort)
		port = defaultPort
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Printf("WARN: DATABASE_URL not set, using default local DSN")
		dbURL = defaultDatabaseURL
	}

	corsEnv := os.Getenv("CORS_ORIGINS")
	if corsEnv == "" {
		logger.Printf("WARN: CORS_ORIGINS not set, using default local origins")
		corsEnv = defaultCORSOrigins
	}

	sessionTTL := defaultSessionTTL
	sessionTTLRaw := os.Getenv("SESSION_TTL")
	if sessionTTLRaw == "" {
		logger.Printf("WARN: SESSION_TTL not set, using default %s", defaultSessionTTL)
	} else if parsed, err := time.ParseDuration(sessionTTLRaw); err != nil {
		logger.Printf("WARN: SESSION_TTL invalid, using default %s", defaultSessionTTL)
	} else {
		sessionTTL = parsed
	}

	cookieSecure := config.ResolveSessionCookieSecure(logger)
	allowRegister := config.ResolveAllowPublicRegister(logger)

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(startupCtx, dbURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(startupCtx); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	if err := migrations.Apply(startupCtx, pool); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	holdRepo := postgres.NewHoldRepository(pool)
	holdSvc := app.NewHoldService(holdRepo, clock.NewSystem())
	orderRepo := postgres.NewOrderRepository(pool)
	orderSvc := app.NewOrderService(orderRepo, clock.NewSystem())
	adminRepo := postgres.NewAdminRepository(pool)
	adminSvc := app.NewAdminService(adminRepo, clock.NewSystem())
	authRepo := postgres.NewAuthRepository(pool)
	authSvc := app.NewAuthService(authRepo, clock.NewSystem(), sessionTTL)

	cookieCfg := transporthttp.SessionCookieConfig{
		Name:     defaultSessionCookieName,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   cookieSecure,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", transporthttp.HealthHandler)
	mux.Handle("/events", transporthttp.HandleEvents(adminSvc))
	mux.Handle("/events/", transporthttp.HandleEventZones(adminSvc))
	mux.Handle("/auth/register", transporthttp.HandleRegister(authSvc, cookieCfg, allowRegister))
	mux.Handle("/auth/login", transporthttp.HandleLogin(authSvc, cookieCfg))
	mux.Handle("/auth/logout", transporthttp.HandleLogout(authSvc, cookieCfg))
	mux.Handle("/auth/password", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.HandleChangePassword(authSvc, cookieCfg)))
	mux.Handle("/me", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.HandleMe()))
	mux.Handle("/holds", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.HandleCreateHold(holdSvc)))
	mux.Handle("/holds/", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.HandleConfirmHold(orderSvc)))
	mux.Handle("/orders", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.HandleOrders(orderSvc)))
	mux.Handle("/admin/events", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.RequireAdmin(transporthttp.HandleAdminEvents(adminSvc))))
	mux.Handle("/admin/events/", transporthttp.RequireAuth(authSvc, cookieCfg, transporthttp.RequireAdmin(transporthttp.HandleAdminZones(adminSvc))))
	mux.Handle("/", transporthttp.NotFoundHandler())

	corsOrigins := parseCSV(corsEnv)
	handler := transporthttp.RequestID(transporthttp.RequestLogger(transporthttp.CORS(corsOrigins, mux), logger))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("api listening on :%s", port)

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- server.ListenAndServe()
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-srvErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	case <-stopCtx.Done():
		log.Printf("shutdown signal received, stopping server")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server shutdown error: %v", err)
	}
	log.Printf("server stopped")
}

func parseCSV(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
