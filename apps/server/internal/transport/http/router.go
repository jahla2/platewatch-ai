package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/observability"
	"github.com/jahla2/platewatch-ai/apps/server/internal/ports"
)

const maxRequestBodyBytes = 1 << 20

type HandlerConfig struct {
	WebOrigin            string
	RequestTimeout       time.Duration
	PublicRequestsPerMin int
	InternalEventsPerMin int
	AdminRequestsPerMin  int
}

type Handler struct {
	detectionService *application.DetectionService
	watchlistService *application.WatchlistService
	subscriber       ports.DetectionSubscriber
	readiness        ports.ReadinessChecker
	webOrigin        string
	internalAuth     *BearerAuthorizer
	adminAuth        *BearerAuthorizer
	logger           *slog.Logger
	metrics          *observability.Metrics
	requestTimeout   time.Duration
	publicLimiter    *FixedWindowRateLimiter
	internalLimiter  *FixedWindowRateLimiter
	adminLimiter     *FixedWindowRateLimiter
}

func NewHandler(
	detectionService *application.DetectionService,
	watchlistService *application.WatchlistService,
	subscriber ports.DetectionSubscriber,
	readiness ports.ReadinessChecker,
	internalAuth *BearerAuthorizer,
	adminAuth *BearerAuthorizer,
	logger *slog.Logger,
	metrics *observability.Metrics,
	config HandlerConfig,
) *Handler {
	return &Handler{
		detectionService: detectionService,
		watchlistService: watchlistService,
		subscriber:       subscriber,
		readiness:        readiness,
		webOrigin:        config.WebOrigin,
		internalAuth:     internalAuth,
		adminAuth:        adminAuth,
		logger:           logger,
		metrics:          metrics,
		requestTimeout:   config.RequestTimeout,
		publicLimiter: NewFixedWindowRateLimiter(
			config.PublicRequestsPerMin,
			time.Minute,
		),
		internalLimiter: NewFixedWindowRateLimiter(
			config.InternalEventsPerMin,
			time.Minute,
		),
		adminLimiter: NewFixedWindowRateLimiter(
			config.AdminRequestsPerMin,
			time.Minute,
		),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.Handle(
		"GET /readyz",
		TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.ready)),
	)
	mux.HandleFunc("GET /metrics", h.metricsHandler)

	mux.Handle(
		"GET /api/v1/detections",
		RateLimitMiddleware(
			h.publicLimiter,
			h.metrics,
			TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.listDetections)),
		),
	)
	mux.Handle(
		"GET /api/v1/events/stream",
		RateLimitMiddleware(
			h.publicLimiter,
			h.metrics,
			http.HandlerFunc(h.streamEvents),
		),
	)
	mux.Handle(
		"POST /internal/v1/detections",
		RateLimitMiddleware(
			h.internalLimiter,
			h.metrics,
			h.internalAuth.Middleware(
				TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.createDetection)),
			),
		),
	)
	mux.Handle(
		"GET /api/v1/watchlist",
		RateLimitMiddleware(
			h.adminLimiter,
			h.metrics,
			h.adminAuth.Middleware(
				TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.listWatchlist)),
			),
		),
	)
	mux.Handle(
		"PUT /api/v1/watchlist/{plate}",
		RateLimitMiddleware(
			h.adminLimiter,
			h.metrics,
			h.adminAuth.Middleware(
				TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.upsertWatchlist)),
			),
		),
	)
	mux.Handle(
		"DELETE /api/v1/watchlist/{plate}",
		RateLimitMiddleware(
			h.adminLimiter,
			h.metrics,
			h.adminAuth.Middleware(
				TimeoutMiddleware(h.requestTimeout, http.HandlerFunc(h.deleteWatchlist)),
			),
		),
	)

	handler := h.securityHeaders(h.withCORS(mux))
	handler = RecoveryMiddleware(h.logger, handler)
	return ObservabilityMiddleware(h.logger, h.metrics, handler)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.readiness.Ping(r.Context()); err != nil {
		h.logger.Error("readiness_check_failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if err := h.metrics.WritePrometheus(w); err != nil {
		h.logger.Error("metrics_write_failed", "error", err)
	}
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

	event, created, err := h.detectionService.Create(
		r.Context(),
		input,
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		h.writeApplicationError(w, r, err)
		return
	}

	if created {
		h.metrics.IncDetectionCreated()
		writeJSON(w, http.StatusCreated, event)
		return
	}

	h.metrics.IncDetectionReplay()
	w.Header().Set("Idempotent-Replayed", "true")
	writeJSON(w, http.StatusOK, event)
}

func (h *Handler) listDetections(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"), 50, 100)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	page, err := h.detectionService.List(
		r.Context(),
		limit,
		r.URL.Query().Get("cursor"),
	)
	if err != nil {
		h.writeApplicationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) listWatchlist(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"), 100, 200)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	entries, err := h.watchlistService.List(r.Context(), limit)
	if err != nil {
		h.writeApplicationError(w, r, err)
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
	input.PlateText = r.PathValue("plate")

	entry, err := h.watchlistService.Upsert(r.Context(), input)
	if err != nil {
		h.writeApplicationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) deleteWatchlist(w http.ResponseWriter, r *http.Request) {
	if err := h.watchlistService.Delete(r.Context(), r.PathValue("plate")); err != nil {
		h.writeApplicationError(w, r, err)
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
	h.metrics.AddSSEConnection(1)
	defer func() {
		h.metrics.AddSSEConnection(-1)
		unsubscribe()
	}()

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
				h.logger.Error("sse_encode_failed", "error", err)
				continue
			}
			if _, err := fmt.Fprintf(
				w,
				"id: %s\nevent: detection\ndata: %s\n\n",
				event.ID,
				payload,
			); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) writeApplicationError(w http.ResponseWriter, r *http.Request, err error) {
	var validationError application.ValidationError
	switch {
	case errors.As(err, &validationError):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": validationError.Error(),
		})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "request timed out"})
	default:
		h.logger.Error(
			"request_failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
	}
}

func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == h.webOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
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
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if r.URL.Path != "/metrics" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func parseLimit(value string, fallback int, maximum int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maximum {
		return 0, fmt.Errorf("limit must be between 1 and %d", maximum)
	}
	return limit, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
