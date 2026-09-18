package httptransport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

type Handler struct {
	service    *application.DetectionService
	subscriber ports.DetectionSubscriber
	webOrigin  string
}

func NewHandler(
	service *application.DetectionService,
	subscriber ports.DetectionSubscriber,
	webOrigin string,
) *Handler {
	return &Handler{
		service: service,
		subscriber: subscriber,
		webOrigin: webOrigin,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /api/v1/detections", h.listDetections)
	mux.HandleFunc("POST /api/v1/detections", h.createDetection)
	mux.HandleFunc("GET /api/v1/events/stream", h.streamEvents)
	return h.withCORS(mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createDetection(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var input application.CreateDetectionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	event, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *Handler) listDetections(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.service.List(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list detections"})
		return
	}
	writeJSON(w, http.StatusOK, events)
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
