package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// stubRT is a RoundTripper that returns a fixed status/error.
type stubRT struct {
	status int
	err    error
}

func (s stubRT) RoundTrip(*http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{StatusCode: s.status, Body: http.NoBody, Header: make(http.Header)}, nil
}

func TestBreakerTransportOpensOn5xx(t *testing.T) {
	cb := NewCircuitBreaker("t", 3, time.Second)
	tr := breakerTransport{base: stubRT{status: 500}, cb: cb}
	req := httptest.NewRequest(http.MethodGet, "http://downstream/x", nil)

	for i := 0; i < 3; i++ {
		_, _ = tr.RoundTrip(req) // 5xx counts as failure
	}
	if cb.State() != "open" {
		t.Fatalf("expected breaker open after three 5xx, got %s", cb.State())
	}

	// Once open, RoundTrip must short-circuit with ErrCircuitOpen.
	_, err := tr.RoundTrip(req)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen when open, got %v", err)
	}
}

func TestBreakerTransportPassesThrough2xx(t *testing.T) {
	cb := NewCircuitBreaker("t", 3, time.Second)
	tr := breakerTransport{base: stubRT{status: 200}, cb: cb}
	req := httptest.NewRequest(http.MethodGet, "http://downstream/x", nil)

	resp, err := tr.RoundTrip(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200 pass-through, got status=%v err=%v", resp, err)
	}
	if cb.State() != "closed" {
		t.Fatalf("expected breaker to stay closed on success, got %s", cb.State())
	}
}
