package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/adapters/postgres"
	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/realtime"
	httptransport "github.com/jahla2/platewatch-ai/apps/server/internal/transport/http"
)

func main() {
	port := env("PLATEWATCH_SERVER_PORT", "8080")
	webOrigin := env("PLATEWATCH_WEB_ORIGIN", "http://localhost:3000")
	databaseURL := requiredEnv("DATABASE_URL")
	internalToken := requiredEnv("PLATEWATCH_INTERNAL_TOKEN")
	adminToken := requiredEnv("PLATEWATCH_ADMIN_TOKEN")

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := postgres.Open(startupCtx, databaseURL)
	if err != nil {
		log.Fatalf("database startup failed: %v", err)
	}
	defer store.Close()

	if err := seedWatchlist(startupCtx, store, os.Getenv("PLATEWATCH_FLAGGED_PLATES")); err != nil {
		log.Fatalf("watchlist seed failed: %v", err)
	}

	broker := realtime.NewBroker()
	detectionService := application.NewDetectionService(store, store, broker)
	watchlistService := application.NewWatchlistService(store)
	handler := httptransport.NewHandler(
		detectionService,
		watchlistService,
		broker,
		webOrigin,
		httptransport.NewBearerAuthorizer(internalToken),
		httptransport.NewBearerAuthorizer(adminToken),
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("PlateWatch server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
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

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}
