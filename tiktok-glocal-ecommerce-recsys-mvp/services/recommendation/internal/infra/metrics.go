package infra

// In-process observability, standard library only (mirrors what a Prometheus
// client would export, without the dependency). Every service embeds the same
// Metrics registry:
//   - request counters (total + per status class + per route)
//   - a fixed-bucket latency histogram with p50/p90/p99 estimation
//   - named app counters (cache hits, rate-limited requests, ...)
//   - string gauges (circuit-breaker states), read lazily at scrape time
// Exposed as Prometheus text on /metrics and machine-friendly JSON on /metricsz.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// latencyBucketsMs are the histogram upper bounds, in milliseconds.
var latencyBucketsMs = []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500}

const maxRouteLabels = 50

type Metrics struct {
	service string
	start   time.Time

	mu            sync.Mutex
	requestsTotal int64
	statusClass   map[string]int64
	routes        map[string]int64
	sumMs         float64
	maxMs         float64
	bucketCounts  []int64 // one per latencyBucketsMs entry, plus +Inf at the end
	counters      map[string]int64
	gauges        map[string]func() string
}

func NewMetrics(service string) *Metrics {
	return &Metrics{
		service:      service,
		start:        time.Now(),
		statusClass:  map[string]int64{},
		routes:       map[string]int64{},
		bucketCounts: make([]int64, len(latencyBucketsMs)+1),
		counters:     map[string]int64{},
		gauges:       map[string]func() string{},
	}
}

// ObserveRequest records one finished HTTP request.
func (m *Metrics) ObserveRequest(route string, status int, durMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestsTotal++
	m.statusClass[fmt.Sprintf("%dxx", status/100)]++
	if len(m.routes) < maxRouteLabels {
		m.routes[route]++
	} else {
		m.routes["other"]++
	}
	m.sumMs += durMs
	if durMs > m.maxMs {
		m.maxMs = durMs
	}
	idx := len(latencyBucketsMs)
	for i, ub := range latencyBucketsMs {
		if durMs <= ub {
			idx = i
			break
		}
	}
	m.bucketCounts[idx]++
	if status == http.StatusTooManyRequests {
		m.counters["rate_limited_total"]++
	}
}

// Inc increments a named application counter.
func (m *Metrics) Inc(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// SetGauge registers a lazily-evaluated string gauge (e.g. a breaker state).
func (m *Metrics) SetGauge(name string, fn func() string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = fn
}

// RequestsTotal returns the number of requests observed so far.
func (m *Metrics) RequestsTotal() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestsTotal
}

// quantileLocked estimates a quantile from the histogram by linear
// interpolation inside the winning bucket. Callers must hold m.mu.
func (m *Metrics) quantileLocked(q float64) float64 {
	total := int64(0)
	for _, c := range m.bucketCounts {
		total += c
	}
	if total == 0 {
		return 0
	}
	rank := q * float64(total)
	cum := int64(0)
	for i, c := range m.bucketCounts {
		cum += c
		if float64(cum) >= rank {
			lower := 0.0
			if i > 0 {
				lower = latencyBucketsMs[i-1]
			}
			upper := m.maxMs // +Inf bucket: cap at the observed max
			if i < len(latencyBucketsMs) {
				upper = latencyBucketsMs[i]
			}
			if c == 0 {
				return upper
			}
			within := rank - float64(cum-c)
			return lower + (upper-lower)*math.Min(1, within/float64(c))
		}
	}
	return m.maxMs
}

// Snapshot returns the JSON document served on /metricsz.
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	avg := 0.0
	if m.requestsTotal > 0 {
		avg = m.sumMs / float64(m.requestsTotal)
	}
	status := map[string]int64{}
	for k, v := range m.statusClass {
		status[k] = v
	}
	routes := map[string]int64{}
	for k, v := range m.routes {
		routes[k] = v
	}
	counters := map[string]int64{}
	for k, v := range m.counters {
		counters[k] = v
	}
	gauges := map[string]string{}
	for k, fn := range m.gauges {
		gauges[k] = fn()
	}
	return map[string]interface{}{
		"service":        m.service,
		"uptime_s":       round2(time.Since(m.start).Seconds()),
		"requests_total": m.requestsTotal,
		"status":         status,
		"latency_ms": map[string]float64{
			"avg": round2(avg),
			"p50": round2(m.quantileLocked(0.50)),
			"p90": round2(m.quantileLocked(0.90)),
			"p99": round2(m.quantileLocked(0.99)),
			"max": round2(m.maxMs),
		},
		"routes":   routes,
		"counters": counters,
		"gauges":   gauges,
	}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// WritePrometheus renders the registry in Prometheus text exposition format.
func (m *Metrics) WritePrometheus(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Fprintf(w, "# HELP http_requests_total Total HTTP requests handled.\n")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
	fmt.Fprintf(w, "http_requests_total{service=%q} %d\n", m.service, m.requestsTotal)
	for _, class := range sortedKeys(m.statusClass) {
		fmt.Fprintf(w, "http_requests_class_total{service=%q,class=%q} %d\n", m.service, class, m.statusClass[class])
	}

	fmt.Fprintf(w, "# HELP http_request_duration_ms HTTP request latency histogram (ms).\n")
	fmt.Fprintf(w, "# TYPE http_request_duration_ms histogram\n")
	cum := int64(0)
	for i, ub := range latencyBucketsMs {
		cum += m.bucketCounts[i]
		fmt.Fprintf(w, "http_request_duration_ms_bucket{service=%q,le=\"%g\"} %d\n", m.service, ub, cum)
	}
	cum += m.bucketCounts[len(latencyBucketsMs)]
	fmt.Fprintf(w, "http_request_duration_ms_bucket{service=%q,le=\"+Inf\"} %d\n", m.service, cum)
	fmt.Fprintf(w, "http_request_duration_ms_sum{service=%q} %.3f\n", m.service, m.sumMs)
	fmt.Fprintf(w, "http_request_duration_ms_count{service=%q} %d\n", m.service, m.requestsTotal)

	for _, name := range sortedKeys(m.counters) {
		fmt.Fprintf(w, "app_counter_total{service=%q,name=%q} %d\n", m.service, name, m.counters[name])
	}
	gaugeNames := make([]string, 0, len(m.gauges))
	for k := range m.gauges {
		gaugeNames = append(gaugeNames, k)
	}
	sort.Strings(gaugeNames)
	for _, name := range gaugeNames {
		fmt.Fprintf(w, "app_state{service=%q,name=%q,state=%q} 1\n", m.service, name, m.gauges[name]())
	}
}

func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---- HTTP plumbing ----

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

var idSegment = regexp.MustCompile(`^(user_\w+|u\d+|\d+|[0-9a-fA-F-]{8,})$`)

// routeLabel collapses dynamic path segments (user ids, uuids, numbers) into
// {id} so the per-route counter map stays low-cardinality.
func routeLabel(r *http.Request) string {
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, s := range segs {
		if idSegment.MatchString(s) {
			segs[i] = "{id}"
		}
	}
	return r.Method + " /" + strings.Join(segs, "/")
}

// Middleware records every request into the registry.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		m.ObserveRequest(routeLabel(r), rec.status, float64(time.Since(start).Microseconds())/1000.0)
	})
}

// Handler serves the Prometheus text endpoint (/metrics).
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m.WritePrometheus(w)
	}
}

// JSONHandler serves the machine-friendly snapshot (/metricsz).
func (m *Metrics) JSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.Snapshot())
	}
}

type ctxKey int

const requestIDKey ctxKey = 0

// RequestIDFromContext returns the request id stored by RequestIDMiddleware, or
// "" when the context carries none. The HTTP repositories use it to propagate
// the trace id on outbound calls to the user/content services.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestIDMiddleware assigns an X-Request-ID when the caller did not send one
// and echoes it on the response, so one id traces a request across services.
// The id also rides the request context so outbound repository calls carry it.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
			r.Header.Set("X-Request-ID", id)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
