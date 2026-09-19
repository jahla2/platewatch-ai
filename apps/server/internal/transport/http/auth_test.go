package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuthorizerRejectsInvalidToken(t *testing.T) {
	auth := NewBearerAuthorizer("secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/internal/v1/detections", nil)
	request.Header.Set("Authorization", "Bearer wrong")
	response := httptest.NewRecorder()

	auth.Middleware(next).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestBearerAuthorizerAcceptsValidToken(t *testing.T) {
	auth := NewBearerAuthorizer("secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/internal/v1/detections", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	auth.Middleware(next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
