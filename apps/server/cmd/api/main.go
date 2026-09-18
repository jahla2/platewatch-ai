package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jahla2/platewatch-ai/apps/server/internal/adapters/memory"
	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/realtime"
	httptransport "github.com/jahla2/platewatch-ai/apps/server/internal/transport/http"
)

func main() {
	port := env("PLATEWATCH_SERVER_PORT", "8080")
	webOrigin := env("PLATEWATCH_WEB_ORIGIN", "http://localhost:3000")
	flaggedPlates := strings.Split(env("PLATEWATCH_FLAGGED_PLATES", "ABC1234,XYZ9876"), ",")

	repository := memory.NewDetectionRepository()
	watchlist := memory.NewWatchlist(flaggedPlates)
	broker := realtime.NewBroker()
	service := application.NewDetectionService(repository, watchlist, broker)
	handler := httptransport.NewHandler(service, broker, webOrigin)

	address := ":" + port
	log.Printf("PlateWatch server listening on %s", address)
	if err := http.ListenAndServe(address, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
