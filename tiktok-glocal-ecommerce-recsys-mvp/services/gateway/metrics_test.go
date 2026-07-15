package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsObserveRequestCountsAndClasses(t *testing.T) {
	m := NewMetrics("test")
	m.ObserveRequest("GET /a", 200, 3)
	m.ObserveRequest("GET /a", 200, 7)
	m.ObserveRequest("GET /b", 404, 1)
	m.ObserveRequest("GET /b", 500, 40)
	m.ObserveRequest("GET /b", 429, 2)

	snap := m.Snapshot()
	if got := snap["requests_total"].(int64); got != 5 {
		t.Fatalf("requests_total = %d, want 5", got)
	}
	status := snap["status"].(map[string]int64)
	if status["2xx"] != 2 || status["4xx"] != 2 || status["5xx"] != 1 {
		t.Fatalf("status classes wrong: %#v", status)
	}
	counters := snap["counters"].(map[string]int64)
	if counters["rate_limited_total"] != 1 {
		t.Fatalf("rate_limited_total = %d, want 1 (429 must be counted)", counters["rate_limited_total"])
	}
	routes := snap["routes"].(map[string]int64)
	if routes["GET /a"] != 2 || routes["GET /b"] != 3 {
		t.Fatalf("route counters wrong: %#v", routes)
	}
}

func TestMetricsQuantileEstimation(t *testing.T) {
	m := NewMetrics("test")
	// 90 fast requests (<=1ms bucket) and 10 slow ones (100-250ms bucket).
	for i := 0; i < 90; i++ {
		m.ObserveRequest("GET /x", 200, 0.5)
	}
	for i := 0; i < 10; i++ {
		m.ObserveRequest("GET /x", 200, 200)
	}
	lat := m.Snapshot()["latency_ms"].(map[string]float64)
	if lat["p50"] > 1 {
		t.Fatalf("p50 = %v, want <= 1 (falls in first bucket)", lat["p50"])
	}
	if lat["p99"] < 100 || lat["p99"] > 250 {
		t.Fatalf("p99 = %v, want inside (100, 250] bucket", lat["p99"])
	}
	if lat["max"] != 200 {
		t.Fatalf("max = %v, want 200", lat["max"])
	}
}

func TestMetricsQuantileEmptyRegistry(t *testing.T) {
	m := NewMetrics("test")
	lat := m.Snapshot()["latency_ms"].(map[string]float64)
	if lat["p50"] != 0 || lat["p99"] != 0 {
		t.Fatalf("empty registry quantiles should be 0, got %#v", lat)
	}
}

func TestMetricsMiddlewareRecordsStatusAndRoute(t *testing.T) {
	m := NewMetrics("test")
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user_123/interactions", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	snap := m.Snapshot()
	if snap["requests_total"].(int64) != 1 {
		t.Fatalf("middleware did not record the request")
	}
	routes := snap["routes"].(map[string]int64)
	if routes["GET /api/v1/users/{id}/interactions"] != 1 {
		t.Fatalf("dynamic id segment not collapsed: %#v", routes)
	}
	if snap["status"].(map[string]int64)["4xx"] != 1 {
		t.Fatalf("status class not recorded: %#v", snap["status"])
	}
}

func TestWritePrometheusFormat(t *testing.T) {
	m := NewMetrics("gw")
	m.ObserveRequest("GET /a", 200, 3)
	m.Inc("cache_hits")
	m.SetGauge("breaker_user", func() string { return "closed" })

	var sb strings.Builder
	m.WritePrometheus(&sb)
	out := sb.String()

	for _, want := range []string{
		`http_requests_total{service="gw"} 1`,
		`http_requests_class_total{service="gw",class="2xx"} 1`,
		`http_request_duration_ms_count{service="gw"} 1`,
		`le="+Inf"`,
		`app_counter_total{service="gw",name="cache_hits"} 1`,
		`app_state{service="gw",name="breaker_user",state="closed"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prometheus output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	var seen string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Request-ID")
	}))

	// No inbound id: one is generated and echoed on the response.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if seen == "" {
		t.Fatal("middleware did not assign X-Request-ID to the request")
	}
	if rec.Header().Get("X-Request-ID") != seen {
		t.Fatal("response X-Request-ID does not match the assigned id")
	}

	// Inbound id is preserved, not replaced.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "trace-abc")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if seen != "trace-abc" || rec2.Header().Get("X-Request-ID") != "trace-abc" {
		t.Fatalf("inbound request id not preserved: seen=%q echoed=%q", seen, rec2.Header().Get("X-Request-ID"))
	}
}
