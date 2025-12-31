package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
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
const shutdownTimeout = 10 * time.Second

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", transporthttp.HealthHandler)
	mux.Handle("/holds", transporthttp.HandleCreateHold(holdSvc))
	mux.Handle("/holds/", transporthttp.HandleConfirmHold(orderSvc))

	handler := transporthttp.RequestLogger(mux, log.Default())

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
		log.Printf("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server shutdown error: %v", err)
	}
}
