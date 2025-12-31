package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	transporthttp "github.com/cimillas/ultimate-ticket/services/api/internal/transport/http"
	"github.com/cimillas/ultimate-ticket/services/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDatabaseURL = "postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable"
const defaultPort = "8080"
const defaultCORSOrigins = "http://localhost:5173,http://127.0.0.1:5173"
const shutdownTimeout = 10 * time.Second

func main() {
	logger := log.Default()
	loadEnvFile(logger)

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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", transporthttp.HealthHandler)
	mux.Handle("/holds", transporthttp.HandleCreateHold(holdSvc))
	mux.Handle("/holds/", transporthttp.HandleConfirmHold(orderSvc))
	mux.Handle("/admin/events", transporthttp.HandleAdminEvents(adminSvc))
	mux.Handle("/admin/events/", transporthttp.HandleAdminZones(adminSvc))
	mux.Handle("/", transporthttp.NotFoundHandler())

	corsOrigins := parseCSV(corsEnv)
	handler := transporthttp.RequestLogger(transporthttp.CORS(corsOrigins, mux), logger)

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

func loadEnvFile(logger *log.Logger) {
	path, err := findEnvFile()
	if err != nil {
		logger.Printf("WARN: failed to locate .env: %v", err)
		return
	}
	if path == "" {
		logger.Printf("WARN: .env not found in current or parent directories")
		return
	}

	file, err := os.Open(path)
	if err != nil {
		logger.Printf("WARN: failed to open %s: %v", path, err)
		return
	}
	if err := parseEnvFile(logger, file); err != nil {
		logger.Printf("WARN: failed to load %s: %v", path, err)
	} else {
		logger.Printf("loaded env from %s", path)
	}
	_ = file.Close()
}

func findEnvFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for i := 0; i < 6; i++ {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil
}

func parseEnvFile(logger *log.Logger, file *os.File) error {
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if lineNum == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		value = trimQuotes(value)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			logger.Printf("WARN: failed to set %s from env file", key)
		}
	}
	return scanner.Err()
}

func trimQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
