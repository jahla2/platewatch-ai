package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/adapters/postgres"
	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/observability"
	"github.com/jahla2/platewatch-ai/apps/server/internal/realtime"
	httptransport "github.com/jahla2/platewatch-ai/apps/server/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := env("PLATEWATCH_SERVER_PORT", "8080")
	webOrigin := env("PLATEWATCH_WEB_ORIGIN", "http://localhost:3000")
	databaseURL := requiredEnv(logger, "DATABASE_URL")
	internalToken := requiredSecret(logger, "PLATEWATCH_INTERNAL_TOKEN", 24)
	adminToken := requiredSecret(logger, "PLATEWATCH_ADMIN_TOKEN", 24)
	operatorToken := requiredSecret(logger, "PLATEWATCH_OPERATOR_TOKEN", 24)
	sessionSecret := requiredSecret(logger, "PLATEWATCH_SESSION_SECRET", 32)

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

	metrics := observability.NewMetrics()
	broker := realtime.NewBroker()
	detectionService := application.NewDetectionService(store, store, broker)
	watchlistService := application.NewWatchlistService(store)
	handler := httptransport.NewHandler(
		detectionService,
		watchlistService,
		broker,
		store,
		httptransport.NewBearerAuthorizer(internalToken),
		httptransport.NewBearerAuthorizer(adminToken),
		httptransport.NewSessionAuthorizer(
			operatorToken,
			sessionSecret,
			envBool("PLATEWATCH_SESSION_SECURE", false),
			time.Duration(envInt("PLATEWATCH_SESSION_HOURS", 12))*time.Hour,
		),
		logger,
		metrics,
		httptransport.HandlerConfig{
			WebOrigin:            webOrigin,
			RequestTimeout:       time.Duration(envInt("PLATEWATCH_API_TIMEOUT_SECONDS", 5)) * time.Second,
			PublicRequestsPerMin: envInt("PLATEWATCH_RATE_LIMIT_PUBLIC_RPM", 120),
			InternalEventsPerMin: envInt("PLATEWATCH_RATE_LIMIT_INTERNAL_RPM", 1200),
			AdminRequestsPerMin:  envInt("PLATEWATCH_RATE_LIMIT_ADMIN_RPM", 60),
			LoginRequestsPerMin:  envInt("PLATEWATCH_RATE_LIMIT_LOGIN_RPM", 10),
		},
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler.Routes(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	runCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	select {
	case <-runCtx.Done():
		logger.Info("server_shutdown_requested")
	case err := <-serverErrors:
		if err != nil {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server_shutdown_failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server_stopped")
}

func seedWatchlist(
	ctx context.Context,
	store *postgres.Store,
	value string,
) error {
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

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func requiredEnv(logger *slog.Logger, key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		logger.Error("configuration_missing", "key", key)
		os.Exit(1)
	}
	return value
}

func requiredSecret(logger *slog.Logger, key string, minimumLength int) string {
	value := requiredEnv(logger, key)
	if len(value) < minimumLength || strings.Contains(strings.ToLower(value), "replace-with") {
		logger.Error(
			"configuration_invalid_secret",
			"key", key,
			"minimum_length", minimumLength,
		)
		os.Exit(1)
	}
	return value
}
