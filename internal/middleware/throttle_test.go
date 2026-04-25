package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginThrottle_AllowsValidKey(t *testing.T) {
	throttle := NewLoginThrottle(3, time.Minute, 5*time.Minute)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := throttle.Wrap("secret", next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestLoginThrottle_RejectsInvalidKey(t *testing.T) {
	throttle := NewLoginThrottle(3, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := throttle.Wrap("secret", next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestLoginThrottle_RejectsMissingAuth(t *testing.T) {
	throttle := NewLoginThrottle(3, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := throttle.Wrap("secret", next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestLoginThrottle_BlocksAfterMaxFailures(t *testing.T) {
	throttle := NewLoginThrottle(3, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := throttle.Wrap("secret", next)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after max failures, got %d", rec.Code)
	}
}

func TestLoginThrottle_DifferentIPsNotBlocked(t *testing.T) {
	throttle := NewLoginThrottle(2, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := throttle.Wrap("secret", next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "5.6.7.8:5678"
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be blocked, got %d", rec.Code)
	}
}

func TestLoginThrottle_SuccessResetsCounter(t *testing.T) {
	throttle := NewLoginThrottle(3, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := throttle.Wrap("secret", next)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for i := 0; i < 3; i++ {
		req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		req.Header.Set("Authorization", "Bearer wrong")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rec.Code)
		}
	}
}

func TestLoginThrottle_BanExpires(t *testing.T) {
	throttle := NewLoginThrottle(2, time.Minute, 100*time.Millisecond)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := throttle.Wrap("secret", next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	time.Sleep(150 * time.Millisecond)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after ban expiry, got %d", rec.Code)
	}
}

func TestLoginThrottle_XForwardedFor(t *testing.T) {
	throttle := NewLoginThrottle(2, time.Minute, 5*time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := throttle.Wrap("secret", next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("X-Forwarded-For IP should be blocked, got %d", rec.Code)
	}
}
