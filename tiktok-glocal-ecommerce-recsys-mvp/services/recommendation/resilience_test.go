package main

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker("t", 3, 50*time.Millisecond)
	failing := func() error { return errors.New("boom") }

	// Three consecutive failures should trip the breaker open.
	for i := 0; i < 3; i++ {
		if err := cb.Execute(failing); err == nil {
			t.Fatalf("attempt %d: expected error", i)
		}
	}
	if cb.State() != "open" {
		t.Fatalf("expected breaker open after 3 failures, got %s", cb.State())
	}

	// While open it must fail fast with ErrCircuitOpen, without calling fn.
	called := false
	err := cb.Execute(func() error { called = true; return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen while open, got %v", err)
	}
	if called {
		t.Fatal("fn must not be called while breaker is open")
	}
}

func TestCircuitBreakerHalfOpenRecovers(t *testing.T) {
	cb := NewCircuitBreaker("t", 1, 20*time.Millisecond)
	_ = cb.Execute(func() error { return errors.New("boom") }) // opens (threshold 1)
	if cb.State() != "open" {
		t.Fatalf("expected open, got %s", cb.State())
	}

	time.Sleep(30 * time.Millisecond) // wait out openTimeout -> half-open probe allowed

	if err := cb.Execute(func() error { return nil }); err != nil {
		t.Fatalf("half-open success probe should pass, got %v", err)
	}
	if cb.State() != "closed" {
		t.Fatalf("expected breaker to close after successful probe, got %s", cb.State())
	}
}

func TestCircuitBreakerHalfOpenReopensOnFailure(t *testing.T) {
	cb := NewCircuitBreaker("t", 1, 20*time.Millisecond)
	_ = cb.Execute(func() error { return errors.New("boom") }) // open
	time.Sleep(30 * time.Millisecond)                          // -> half-open
	_ = cb.Execute(func() error { return errors.New("still down") })
	if cb.State() != "open" {
		t.Fatalf("a failed half-open probe must re-open the breaker, got %s", cb.State())
	}
}

func TestCallResilientRetriesThenSucceeds(t *testing.T) {
	cb := NewCircuitBreaker("t", 5, time.Second)
	attempts := 0
	err := callResilient(cb, 3, time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestCallResilientFailsFastWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker("t", 1, time.Second) // opens after a single failure
	calls := 0
	err := callResilient(cb, 5, time.Millisecond, func() error {
		calls++
		return errors.New("down")
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	// First attempt fails and opens the breaker; remaining attempts fail fast,
	// so fn is invoked exactly once.
	if calls != 1 {
		t.Fatalf("expected fn called once before fail-fast, got %d", calls)
	}
}
