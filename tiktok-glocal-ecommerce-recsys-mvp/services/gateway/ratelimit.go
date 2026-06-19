package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Per-IP token-bucket rate limiter, standard library only. It acts as a basic
// abuse / DDoS safety cap at the API gateway: each client IP gets `rps` tokens
// per second up to a `burst` ceiling; requests beyond that get HTTP 429.

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type IPRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   float64
}

func NewIPRateLimiter(rps, burst float64) *IPRateLimiter {
	return &IPRateLimiter{buckets: make(map[string]*tokenBucket), rps: rps, burst: burst}
}

// allow refills the caller's bucket based on elapsed time and consumes one token
// if available.
func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[ip]
	if !ok {
		// First request from this IP starts with a full bucket minus this token.
		l.buckets[ip] = &tokenBucket{tokens: l.burst - 1, last: now}
		return true
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// Middleware rejects requests that exceed the per-IP rate with HTTP 429.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"code": 429, "message": "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
