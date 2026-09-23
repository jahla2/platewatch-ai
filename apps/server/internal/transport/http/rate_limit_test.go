package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/observability"
)

func TestFixedWindowRateLimiterAllowsThenRejects(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(2, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("second request should be allowed")
	}
	if allowed, retryAfter := limiter.Allow("client"); allowed || retryAfter <= 0 {
		t.Fatalf("third request = %v, %v; want rejected with retry delay", allowed, retryAfter)
	}

	now = now.Add(time.Minute)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestRateLimitMiddlewareReturns429AndRetryAfter(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(1, time.Minute)
	metrics := observability.NewMetrics()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RateLimitMiddleware(limiter, metrics, next)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header is missing")
	}
}
