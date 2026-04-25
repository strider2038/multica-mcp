package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type entry struct {
	count    int
	blocked  bool
	lastSeen time.Time
}

type LoginThrottle struct {
	mu         sync.Mutex
	entries    map[string]*entry
	maxFail    int
	window     time.Duration
	banTime    time.Duration
	cleanEvery time.Duration
	lastClean  time.Time
}

func NewLoginThrottle(maxFail int, window, banTime time.Duration) *LoginThrottle {
	return &LoginThrottle{
		entries:    make(map[string]*entry),
		maxFail:    maxFail,
		window:     window,
		banTime:    banTime,
		cleanEvery: 5 * time.Minute,
		lastClean:  time.Now(),
	}
}

func (t *LoginThrottle) Wrap(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		if t.isBlocked(ip) {
			slog.Warn("request blocked by login throttle", "ip", ip)
			http.Error(w, "too many auth failures; try again later", http.StatusTooManyRequests)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.recordFailure(ip)
			http.Error(w, "unauthorized: missing Authorization header", http.StatusUnauthorized)
			return
		}

		token := auth
		if len(auth) > 7 && auth[:7] == "Bearer " {
			token = auth[7:]
		}

		if token != apiKey {
			t.recordFailure(ip)
			slog.Warn("invalid API key attempt", "ip", ip)
			http.Error(w, "unauthorized: invalid API key", http.StatusUnauthorized)
			return
		}

		t.recordSuccess(ip)
		next.ServeHTTP(w, r)
	})
}

func (t *LoginThrottle) isBlocked(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[ip]
	if !ok {
		return false
	}

	now := time.Now()

	if e.blocked {
		if now.Sub(e.lastSeen) > t.banTime {
			delete(t.entries, ip)
			return false
		}
		return true
	}

	return false
}

func (t *LoginThrottle) recordFailure(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.maybeClean()

	now := time.Now()
	e, ok := t.entries[ip]
	if !ok {
		e = &entry{}
		t.entries[ip] = e
	}

	if now.Sub(e.lastSeen) > t.window {
		e.count = 0
	}

	e.count++
	e.lastSeen = now

	if e.count >= t.maxFail {
		e.blocked = true
		slog.Warn("IP blocked by login throttle", "ip", ip, "failures", e.count, "ban_duration", t.banTime)
	}
}

func (t *LoginThrottle) recordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, ip)
}

func (t *LoginThrottle) maybeClean() {
	now := time.Now()
	if now.Sub(t.lastClean) < t.cleanEvery {
		return
	}
	t.lastClean = now
	for ip, e := range t.entries {
		if now.Sub(e.lastSeen) > t.banTime && now.Sub(e.lastSeen) > t.window {
			delete(t.entries, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
