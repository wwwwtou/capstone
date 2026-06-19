package main

import (
	"errors"
	"sync"
	"time"
)

// This file implements fault-tolerance primitives (circuit breaker + retry with
// backoff) using only the standard library, matching the project's
// no-external-dependency convention. They guard the recommendation service's
// synchronous calls to the user and content microservices so a slow or failing
// downstream cannot cascade into the core recommendation path.

// ErrCircuitOpen is returned when a call is rejected because the breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

// CircuitBreaker is a minimal three-state breaker (closed -> open -> half-open).
// After maxFailures consecutive failures it opens and fails fast; after
// openTimeout it allows a single half-open probe to decide whether to close.
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

// allow reports whether a call may proceed and transitions open -> half-open
// once the cooldown has elapsed.
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
	// A failed half-open probe, or hitting the threshold, (re)opens the breaker.
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
// attempt guarded by the breaker. It fails fast (no further retries) the moment
// the breaker reports open, so a downstream outage is not hammered.
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
			time.Sleep(base * time.Duration(1<<i)) // 1x, 2x, 4x ... backoff
		}
	}
	return err
}
