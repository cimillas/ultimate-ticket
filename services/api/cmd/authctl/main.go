package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/config"
	"github.com/cimillas/ultimate-ticket/services/api/internal/storage/postgres"
	"github.com/cimillas/ultimate-ticket/services/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDatabaseURL = "postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable"
	adminEnvUsername   = "ADMIN_USERNAME"
	adminEnvEmail      = "ADMIN_EMAIL"
	adminEnvPassword   = "ADMIN_PASSWORD"
	appEnvKey          = "APP_ENV"
	confirmKey         = "CONFIRM"
)

func main() {
	logger := log.Default()
	config.LoadEnvFile(logger)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "bootstrap-admin":
		runBootstrap(logger)
	case "reset-auth":
		runReset(logger)
	default:
		usage()
		os.Exit(2)
	}
}

func runBootstrap(logger *log.Logger) {
	ctx := context.Background()
	pool := connectDB(ctx, logger)
	defer pool.Close()

	adminUsername := requireEnv(adminEnvUsername)
	adminEmail := requireEnv(adminEnvEmail)
	adminPassword := requireEnv(adminEnvPassword)

	authSvc := newAuthService(pool)
	if err := authSvc.BootstrapAdmin(ctx, adminUsername, adminEmail, adminPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}
	log.Printf("admin bootstrapped")
}

func runReset(logger *log.Logger) {
	if os.Getenv(appEnvKey) != "local" {
		log.Fatalf("reset-auth allowed only when %s=local", appEnvKey)
	}
	if os.Getenv(confirmKey) != "YES" {
		log.Fatalf("reset-auth requires %s=YES", confirmKey)
	}

	ctx := context.Background()
	pool := connectDB(ctx, logger)
	defer pool.Close()

	adminUsername := requireEnv(adminEnvUsername)
	adminEmail := requireEnv(adminEnvEmail)
	adminPassword := requireEnv(adminEnvPassword)

	authSvc := newAuthService(pool)
	if err := authSvc.ResetAuth(ctx, adminUsername, adminEmail, adminPassword); err != nil {
		log.Fatalf("reset auth: %v", err)
	}
	log.Printf("auth reset complete")
}

func connectDB(ctx context.Context, logger *log.Logger) *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Printf("WARN: DATABASE_URL not set, using default local DSN")
		dbURL = defaultDatabaseURL
	}

	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(startupCtx, dbURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	if err := pool.Ping(startupCtx); err != nil {
		pool.Close()
		log.Fatalf("db ping: %v", err)
	}
	if err := migrations.Apply(startupCtx, pool); err != nil {
		pool.Close()
		log.Fatalf("apply migrations: %v", err)
	}
	return pool
}

func newAuthService(pool *pgxpool.Pool) *app.AuthService {
	repo := postgres.NewAuthRepository(pool)
	return app.NewAuthService(repo, clock.NewSystem(), time.Hour)
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("%s must be set", key)
	}
	return val
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  authctl bootstrap-admin")
	fmt.Println("  authctl reset-auth")
	fmt.Println("  reset-auth requires APP_ENV=local and CONFIRM=YES")
}
