package httptransport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

const maxRequestBodyBytes = 1 << 20

type Handler struct {
	detectionService *application.DetectionService
	watchlistService *application.WatchlistService
	subscriber       ports.DetectionSubscriber
	webOrigin        string
	internalAuth     *BearerAuthorizer
	adminAuth        *BearerAuthorizer
}

func NewHandler(
	detectionService *application.DetectionService,
	watchlistService *application.WatchlistService,
	subscriber ports.DetectionSubscriber,
	webOrigin string,
	internalAuth *BearerAuthorizer,
	adminAuth *BearerAuthorizer,
) *Handler {
	return &Handler{
		detectionService: detectionService,
		watchlistService: watchlistService,
		subscriber:       subscriber,
		webOrigin:        webOrigin,
		internalAuth:     internalAuth,
		adminAuth:        adminAuth,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /api/v1/detections", h.listDetections)
	mux.HandleFunc("GET /api/v1/events/stream", h.streamEvents)
	mux.Handle(
		"POST /internal/v1/detections",
		h.internalAuth.Middleware(http.HandlerFunc(h.createDetection)),
	)
	mux.Handle(
		"GET /api/v1/watchlist",
		h.adminAuth.Middleware(http.HandlerFunc(h.listWatchlist)),
	)
	mux.Handle(
		"PUT /api/v1/watchlist/{plate}",
		h.adminAuth.Middleware(http.HandlerFunc(h.upsertWatchlist)),
	)
	mux.Handle(
		"DELETE /api/v1/watchlist/{plate}",
		h.adminAuth.Middleware(http.HandlerFunc(h.deleteWatchlist)),
	)
	return h.securityHeaders(h.withCORS(mux))
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createDetection(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var input application.CreateDetectionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	event, err := h.detectionService.Create(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *Handler) listDetections(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.detectionService.List(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list detections"})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) listWatchlist(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.watchlistService.List(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list watchlist"})
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) upsertWatchlist(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var input application.UpsertWatchlistInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}
	input.Plate = r.PathValue("plate")

	entry, err := h.watchlistService.Upsert(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) deleteWatchlist(w http.ResponseWriter, r *http.Request) {
	if err := h.watchlistService.Delete(r.Context(), r.PathValue("plate")); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "watchlist entry not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events, unsubscribe := h.subscriber.Subscribe()
	defer unsubscribe()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: detection\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", h.webOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
