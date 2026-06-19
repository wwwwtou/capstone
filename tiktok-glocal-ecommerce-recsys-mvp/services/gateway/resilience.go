package main

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// Standard-library circuit breaker used to protect the gateway's reverse-proxy
// calls to each downstream microservice. When a downstream starts failing the
// breaker opens and the gateway returns fast (503) instead of piling requests
// onto a dead service.

// ErrCircuitOpen is returned by the transport when the breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

// CircuitBreaker is a minimal three-state breaker (closed -> open -> half-open).
type CircuitBreaker struct {
	name        string
	maxFailures int
	openTimeout time.Duration

	mu       sync.Mutex
	state    cbState
	failures int
	openedAt time.Time
}

func NewCircuitBreaker(name string, maxFailures int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{name: name, maxFailures: maxFailures, openTimeout: openTimeout}
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == cbOpen {
		if time.Since(cb.openedAt) >= cb.openTimeout {
			cb.state = cbHalfOpen
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = cbClosed
}

func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.state == cbHalfOpen || cb.failures >= cb.maxFailures {
		cb.state = cbOpen
		cb.openedAt = time.Now()
	}
}

// State returns the current state as a string (for tests/observability).
func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// breakerTransport wraps an http.RoundTripper with a circuit breaker. A
// transport error or a 5xx response counts as a failure; once the breaker is
// open it short-circuits with ErrCircuitOpen so the reverse proxy's ErrorHandler
// can return 503 immediately.
type breakerTransport struct {
	base http.RoundTripper
	cb   *CircuitBreaker
}

func (t breakerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !t.cb.allow() {
		return nil, ErrCircuitOpen
	}
	resp, err := t.base.RoundTrip(r)
	if err != nil || (resp != nil && resp.StatusCode >= 500) {
		t.cb.onFailure()
		return resp, err
	}
	t.cb.onSuccess()
	return resp, nil
}
