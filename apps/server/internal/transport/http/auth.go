package httptransport

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const operatorSessionCookie = "platewatch_session"

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
		if !constantTimeEqual(provided, a.token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

type SessionAuthorizer struct {
	operatorToken string
	sessionSecret string
	secureCookie  bool
	sessionTTL    time.Duration
	now           func() time.Time
}

func NewSessionAuthorizer(
	operatorToken string,
	sessionSecret string,
	secureCookie bool,
	sessionTTL time.Duration,
) *SessionAuthorizer {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	return &SessionAuthorizer{
		operatorToken: strings.TrimSpace(operatorToken),
		sessionSecret: strings.TrimSpace(sessionSecret),
		secureCookie:  secureCookie,
		sessionTTL:    sessionTTL,
		now:           time.Now,
	}
}

func (a *SessionAuthorizer) Authenticate(w http.ResponseWriter, providedToken string) bool {
	if !a.configured() || !constantTimeEqual(strings.TrimSpace(providedToken), a.operatorToken) {
		return false
	}

	expiresAt := a.now().Add(a.sessionTTL).UTC()
	http.SetCookie(w, &http.Cookie{
		Name:     operatorSessionCookie,
		Value:    a.sessionValue(expiresAt),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(a.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
	return true
}

func (a *SessionAuthorizer) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     operatorSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (a *SessionAuthorizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.configured() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "operator authentication is not configured",
			})
			return
		}

		cookie, err := r.Cookie(operatorSessionCookie)
		if err != nil || !a.validSession(cookie.Value) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *SessionAuthorizer) configured() bool {
	return a.operatorToken != "" && a.sessionSecret != ""
}

func (a *SessionAuthorizer) sessionValue(expiresAt time.Time) string {
	expiry := strconv.FormatInt(expiresAt.Unix(), 10)
	signature := a.sign(expiry)
	return expiry + "." + signature
}

func (a *SessionAuthorizer) validSession(value string) bool {
	expiryText, signature, found := strings.Cut(value, ".")
	if !found || expiryText == "" || signature == "" {
		return false
	}

	expiryUnix, err := strconv.ParseInt(expiryText, 10, 64)
	if err != nil {
		return false
	}
	expiresAt := time.Unix(expiryUnix, 0)
	if !a.now().Before(expiresAt) {
		return false
	}

	return constantTimeEqual(signature, a.sign(expiryText))
}

func (a *SessionAuthorizer) sign(expiry string) string {
	mac := hmac.New(sha256.New, []byte(a.sessionSecret))
	_, _ = mac.Write([]byte(a.operatorToken))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(expiry))
	return hex.EncodeToString(mac.Sum(nil))
}

func constantTimeEqual(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
