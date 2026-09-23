package httptransport

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/observability"
)

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

type FixedWindowRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
	now     func() time.Time
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &FixedWindowRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateLimitEntry),
		now:     time.Now,
	}
}

func (l *FixedWindowRateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, found := l.entries[key]
	if !found || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = rateLimitEntry{windowStart: now, count: 1}
		l.cleanup(now)
		return true, 0
	}

	if entry.count >= l.limit {
		return false, l.window - now.Sub(entry.windowStart)
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (l *FixedWindowRateLimiter) cleanup(now time.Time) {
	if len(l.entries) < 4096 {
		return
	}
	for key, entry := range l.entries {
		if now.Sub(entry.windowStart) >= 2*l.window {
			delete(l.entries, key)
		}
	}
}

func RateLimitMiddleware(
	limiter *FixedWindowRateLimiter,
	metrics *observability.Metrics,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		allowed, retryAfter := limiter.Allow(key)
		if !allowed {
			metrics.IncRateLimited()
			seconds := int(retryAfter.Round(time.Second).Seconds())
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	peerHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peerHost = r.RemoteAddr
	}
	peerIP := net.ParseIP(peerHost)

	// Only trust proxy headers when the immediate peer is local/private. The Go
	// API is normally reached through the internal Nginx container.
	if peerIP != nil && (peerIP.IsPrivate() || peerIP.IsLoopback()) {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			first, _, _ := strings.Cut(forwarded, ",")
			if candidate := strings.TrimSpace(first); net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}

	if peerHost != "" {
		return peerHost
	}
	return "unknown"
}
