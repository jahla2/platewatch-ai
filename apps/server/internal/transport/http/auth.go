package httptransport

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type BearerAuthorizer struct {
	token string
}

func NewBearerAuthorizer(token string) *BearerAuthorizer {
	return &BearerAuthorizer{token: strings.TrimSpace(token)}
}

func (a *BearerAuthorizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.token == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "service authentication is not configured",
			})
			return
		}

		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		provided := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
