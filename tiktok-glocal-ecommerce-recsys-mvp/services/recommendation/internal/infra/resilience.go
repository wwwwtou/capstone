// Package infra holds the outward-facing adapters that implement the domain
// repository ports: HTTP clients to the user/content services and the Postgres
// config store, plus the fault-tolerance primitives that guard them.
package infra

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when a call is rejected because the breaker is open.
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

// State returns the current breaker state as a string (for tests/observability).
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

// Execute runs fn under the breaker, recording success/failure. It returns
// ErrCircuitOpen without invoking fn when the breaker is open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allow() {
		return ErrCircuitOpen
	}
	err := fn()
	if err != nil {
		cb.onFailure()
		return err
	}
	cb.onSuccess()
	return nil
}

// callResilient retries fn up to attempts times with exponential backoff, each
// attempt guarded by the breaker, failing fast the moment the breaker opens.
func callResilient(cb *CircuitBreaker, attempts int, base time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = cb.Execute(fn)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrCircuitOpen) {
			return err
		}
		if i < attempts-1 {
			time.Sleep(base * time.Duration(1<<i))
		}
	}
	return err
}
