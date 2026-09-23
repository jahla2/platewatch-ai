package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsRequestsPastLimit(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	now := time.Unix(1000, 0)
	limiter.now = func() time.Time { return now }

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for index, want := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("request %d status = %d, want %d", index+1, response.Code, want)
		}
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := NewRateLimiter(1, time.Second)
	now := time.Unix(1000, 0)
	limiter.now = func() time.Time { return now }

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(httptest.NewRecorder(), request)

	now = now.Add(2 * time.Second)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
