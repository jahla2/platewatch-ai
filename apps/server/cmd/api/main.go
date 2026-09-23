package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/adapters/postgres"
	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/realtime"
	httptransport "github.com/jahla2/platewatch-ai/apps/server/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := env("PLATEWATCH_SERVER_PORT", "8080")
	webOrigin := env("PLATEWATCH_WEB_ORIGIN", "http://localhost:3000")
	databaseURL := requiredEnv("DATABASE_URL")
	internalToken := requiredEnv("PLATEWATCH_INTERNAL_TOKEN")
	adminToken := requiredEnv("PLATEWATCH_ADMIN_TOKEN")

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := postgres.Open(startupCtx, databaseURL)
	if err != nil {
		logger.Error("database_startup_failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := seedWatchlist(startupCtx, store, os.Getenv("PLATEWATCH_FLAGGED_PLATES")); err != nil {
		logger.Error("watchlist_seed_failed", "error", err)
		os.Exit(1)
	}

	broker := realtime.NewBroker()
	detectionService := application.NewDetectionService(store, store, broker)
	watchlistService := application.NewWatchlistService(store)
	handler := httptransport.NewHandler(
		detectionService,
		watchlistService,
		broker,
		store,
		webOrigin,
		httptransport.NewBearerAuthorizer(internalToken),
		httptransport.NewBearerAuthorizer(adminToken),
		httptransport.NewRateLimiter(envInt("PLATEWATCH_PUBLIC_RATE_LIMIT_PER_MINUTE", 120), time.Minute),
		httptransport.NewRateLimiter(envInt("PLATEWATCH_INTERNAL_RATE_LIMIT_PER_MINUTE", 600), time.Minute),
		httptransport.NewRateLimiter(envInt("PLATEWATCH_ADMIN_RATE_LIMIT_PER_MINUTE", 60), time.Minute),
		logger,
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("server_listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server_stopped", "error", err)
		os.Exit(1)
	}
}

func seedWatchlist(ctx context.Context, store *postgres.Store, value string) error {
	for _, plate := range strings.Split(value, ",") {
		plateText := strings.TrimSpace(plate)
		plateKey := domain.CanonicalizePlateText(plateText)
		if plateText == "" || plateKey == "" {
			continue
		}

		_, err := store.UpsertWatchlist(ctx, domain.WatchlistEntry{
			PlateText: plateText,
			PlateKey:  plateKey,
			Reason:    "development seed",
			Active:    true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		log.Fatalf("%s must be a positive integer", key)
	}
	return parsed
}

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}
