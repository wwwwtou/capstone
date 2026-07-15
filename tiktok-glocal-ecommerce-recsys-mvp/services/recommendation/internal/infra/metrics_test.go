package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDRidesTheContext(t *testing.T) {
	var fromCtx string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromCtx = RequestIDFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	req.Header.Set("X-Request-ID", "trace-xyz")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if fromCtx != "trace-xyz" {
		t.Fatalf("RequestIDFromContext = %q, want trace-xyz", fromCtx)
	}
	if RequestIDFromContext(context.Background()) != "" {
		t.Fatal("empty context should yield empty request id")
	}
}

func TestOutboundCallPropagatesRequestID(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
		w.Write([]byte(`{"user_id":"u1","tags":{}}`))
	}))
	defer upstream.Close()

	repo := NewHTTPProfileRepository(upstream.URL)
	ctx := context.WithValue(context.Background(), requestIDKey, "trace-outbound")
	if _, err := repo.GetProfile(ctx, "u1"); err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if got != "trace-outbound" {
		t.Fatalf("outbound X-Request-ID = %q, want trace-outbound", got)
	}
}

func TestBreakerStateAccessors(t *testing.T) {
	p := NewHTTPProfileRepository("http://127.0.0.1:0")
	c := NewHTTPContentRepository("http://127.0.0.1:0")
	if p.BreakerState() != "closed" || c.BreakerState() != "closed" {
		t.Fatalf("fresh breakers should be closed: %s / %s", p.BreakerState(), c.BreakerState())
	}
}
