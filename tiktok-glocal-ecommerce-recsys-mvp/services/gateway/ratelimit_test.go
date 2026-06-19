package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	// 1 token/sec, burst of 2: the first two requests pass, the third is blocked.
	l := NewIPRateLimiter(1, 2)
	if !l.allow("1.2.3.4") {
		t.Fatal("request 1 should pass (burst)")
	}
	if !l.allow("1.2.3.4") {
		t.Fatal("request 2 should pass (burst)")
	}
	if l.allow("1.2.3.4") {
		t.Fatal("request 3 should be blocked (burst exhausted)")
	}
}

func TestIPRateLimiterRefills(t *testing.T) {
	l := NewIPRateLimiter(100, 1) // 100 tokens/sec, burst 1
	if !l.allow("a") {
		t.Fatal("first request should pass")
	}
	if l.allow("a") {
		t.Fatal("second immediate request should be blocked")
	}
	time.Sleep(20 * time.Millisecond) // ~2 tokens refilled
	if !l.allow("a") {
		t.Fatal("request after refill should pass")
	}
}

func TestIPRateLimiterIsolatesByIP(t *testing.T) {
	l := NewIPRateLimiter(1, 1)
	if !l.allow("ip-a") {
		t.Fatal("ip-a first request should pass")
	}
	if !l.allow("ip-b") {
		t.Fatal("ip-b must have its own bucket and pass")
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	l := NewIPRateLimiter(1, 1)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request expected 429, got %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("429 response should set Retry-After header")
	}
}
