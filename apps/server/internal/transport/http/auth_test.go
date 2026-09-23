package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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


func TestSessionAuthorizerSetsHttpOnlyCookieAndAuthorizes(t *testing.T) {
	auth := NewSessionAuthorizer(
		"operator-token-1234567890123456",
		"session-secret-123456789012345678901234567890",
		false,
		time.Hour,
	)

	login := httptest.NewRecorder()
	if !auth.Authenticate(login, "operator-token-1234567890123456") {
		t.Fatal("Authenticate() = false, want true")
	}

	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
	if cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("SameSite = %v, want Strict", cookies[0].SameSite)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/detections", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()

	auth.Middleware(next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestSessionAuthorizerRejectsInvalidOperatorToken(t *testing.T) {
	auth := NewSessionAuthorizer(
		"operator-token-1234567890123456",
		"session-secret-123456789012345678901234567890",
		false,
		time.Hour,
	)

	response := httptest.NewRecorder()
	if auth.Authenticate(response, "wrong-token") {
		t.Fatal("Authenticate() = true, want false")
	}
}
