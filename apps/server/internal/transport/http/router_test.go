package httptransport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/application"
	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
	"github.com/jahla2/platewatch-ai/apps/server/internal/observability"
	"github.com/jahla2/platewatch-ai/apps/server/internal/realtime"
)

type routerRepository struct {
	mu          sync.Mutex
	events      []domain.DetectionEvent
	idempotency map[string]domain.DetectionEvent
}

func newRouterRepository() *routerRepository {
	return &routerRepository{idempotency: map[string]domain.DetectionEvent{}}
}

func (r *routerRepository) SaveIdempotent(
	_ context.Context,
	key string,
	event domain.DetectionEvent,
) (domain.DetectionEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, found := r.idempotency[key]; found {
		return existing, false, nil
	}
	event.IdempotencyKey = key
	r.idempotency[key] = event
	r.events = append(r.events, event)
	return event, true, nil
}

func (r *routerRepository) List(
	_ context.Context,
	limit int,
	_ *domain.DetectionCursor,
) ([]domain.DetectionEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit > len(r.events) {
		limit = len(r.events)
	}
	return append([]domain.DetectionEvent(nil), r.events[:limit]...), nil
}

type routerWatchlistStore struct{}

func (routerWatchlistStore) IsFlagged(_ context.Context, plate string) (bool, error) {
	return plate == "ABC1234", nil
}

func (routerWatchlistStore) ListWatchlist(
	_ context.Context,
	_ int,
) ([]domain.WatchlistEntry, error) {
	return nil, nil
}

func (routerWatchlistStore) UpsertWatchlist(
	_ context.Context,
	entry domain.WatchlistEntry,
) (domain.WatchlistEntry, error) {
	return entry, nil
}

func (routerWatchlistStore) DeleteWatchlist(_ context.Context, _ string) error {
	return nil
}

type readyFake struct {
	err error
}

func (r readyFake) Ping(_ context.Context) error {
	return r.err
}

func newTestHandler(readinessError error) (*Handler, *routerRepository) {
	repository := newRouterRepository()
	watchlist := routerWatchlistStore{}
	broker := realtime.NewBroker()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	metrics := observability.NewMetrics()

	return NewHandler(
		application.NewDetectionService(repository, watchlist, broker),
		application.NewWatchlistService(watchlist),
		broker,
		readyFake{err: readinessError},
		NewBearerAuthorizer("internal-secret"),
		NewBearerAuthorizer("admin-secret"),
		NewSessionAuthorizer(
			"operator-secret-token-1234567890",
			"session-secret-token-12345678901234567890",
			false,
			time.Hour,
		),
		logger,
		metrics,
		HandlerConfig{
			WebOrigin:            "http://localhost:3000",
			RequestTimeout:       time.Second,
			PublicRequestsPerMin: 1000,
			InternalEventsPerMin: 1000,
			AdminRequestsPerMin:  1000,
			LoginRequestsPerMin:  1000,
		},
	), repository
}

func TestInternalDetectionRequiresAuthentication(t *testing.T) {
	handler, _ := newTestHandler(nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/internal/v1/detections",
		bytes.NewBufferString(`{"camera_id":"CAM-01","track_id":1,"plate_text":"ABC1234","confidence":0.9}`),
	)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestInternalDetectionIsIdempotent(t *testing.T) {
	handler, repository := newTestHandler(nil)
	body := []byte(`{"camera_id":"CAM-01","track_id":1,"plate_text":"ABC1234","confidence":0.9}`)

	send := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost,
			"/internal/v1/detections",
			bytes.NewReader(body),
		)
		request.Header.Set("Authorization", "Bearer internal-secret")
		request.Header.Set("Idempotency-Key", "same-request")
		response := httptest.NewRecorder()
		handler.Routes().ServeHTTP(response, request)
		return response
	}

	first := send()
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusCreated)
	}

	second := send()
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusOK)
	}
	if second.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("missing Idempotent-Replayed header")
	}
	if len(repository.events) != 1 {
		t.Fatalf("saved events = %d, want 1", len(repository.events))
	}
}

func TestInternalDetectionRejectsUnknownJSONField(t *testing.T) {
	handler, _ := newTestHandler(nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/internal/v1/detections",
		bytes.NewBufferString(
			`{"camera_id":"CAM-01","track_id":1,"plate_text":"ABC1234","confidence":0.9,"unknown":true}`,
		),
	)
	request.Header.Set("Authorization", "Bearer internal-secret")
	request.Header.Set("Idempotency-Key", "request-1")
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestReadinessReturns503WhenDatabaseIsUnavailable(t *testing.T) {
	handler, _ := newTestHandler(errors.New("database unavailable"))
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()

	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestOperatorSessionProtectsDetectionHistory(t *testing.T) {
	handler, _ := newTestHandler(nil)

	unauthorized := httptest.NewRecorder()
	handler.Routes().ServeHTTP(
		unauthorized,
		httptest.NewRequest(http.MethodGet, "/api/v1/detections", nil),
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/session",
		bytes.NewBufferString(`{"token":"operator-secret-token-1234567890"}`),
	)
	loginResponse := httptest.NewRecorder()
	handler.Routes().ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusNoContent {
		t.Fatalf("login status = %d, want %d", loginResponse.Code, http.StatusNoContent)
	}

	result := loginResponse.Result()
	cookies := result.Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set a session cookie")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/detections", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, want %d", response.Code, http.StatusOK)
	}
}
