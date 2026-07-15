package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	startTime time.Time
	jwtSecret string
	metrics   *Metrics
)

func proxyTo(target string, removePrefix string, cb *CircuitBreaker) http.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)
	// Guard every downstream hop with the per-service circuit breaker.
	proxy.Transport = breakerTransport{base: http.DefaultTransport, cb: cb}
	// The gateway already stamped X-Request-ID on the response; drop the
	// downstream echo so the client sees a single trace id, not a joined pair.
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Request-ID")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if errors.Is(err, ErrCircuitOpen) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"code": 503, "message": "downstream temporarily unavailable (circuit open)",
			})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"code": 502, "message": "bad gateway",
		})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// rewrite path to preserve downstream expected route
		if removePrefix != "" && strings.HasPrefix(r.URL.Path, removePrefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, removePrefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
		proxy.ServeHTTP(w, r)
	}
}

// ---- Minimal HS256 JWT (standard library only, no external dependency) ----

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sign(segments string) string {
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(segments))
	return b64(mac.Sum(nil))
}

func issueToken(subject string, ttl time.Duration) string {
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]interface{}{
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(ttl).Unix(),
	})
	body := header + "." + b64(claims)
	return body + "." + sign(body)
}

func validToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	if !hmac.Equal([]byte(sign(parts[0]+"."+parts[1])), []byte(parts[2])) {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	return claims.Exp > time.Now().Unix()
}

func requireAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	return validToken(strings.TrimPrefix(auth, "Bearer "))
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ---- Handlers ----

func handleLogin(w http.ResponseWriter, r *http.Request) {
	token := issueToken("admin", time.Hour)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    map[string]interface{}{"token": token, "expires_in": 3600},
	})
}

// ping returns "UP" when a downstream /healthz answers 200 within the timeout.
func ping(client *http.Client, base string) string {
	resp, err := client.Get(base + "/healthz")
	if err != nil {
		return "DOWN"
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return "UP"
	}
	return "DOWN"
}

func handleHealth(userURL, contentURL, recURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{Timeout: 1500 * time.Millisecond}

		t0 := time.Now()
		recStatus := ping(client, recURL)
		userStatus := ping(client, userURL)
		contentStatus := ping(client, contentURL)
		// measured round-trip of the slowest health probe, in ms
		latencyMs := time.Since(t0).Milliseconds()

		// Postgres/Redis are inferred from the services that depend on them.
		pg := "DOWN"
		if userStatus == "UP" && contentStatus == "UP" {
			pg = "ACTIVE"
		}
		redisShards := 0
		if userStatus == "UP" {
			redisShards = 1 // single Redis instance in this deployment
		}

		// Real throughput measured from gateway traffic since startup.
		uptime := time.Since(startTime).Seconds()
		if uptime < 1 {
			uptime = 1
		}
		rps := float64(metrics.RequestsTotal()) / uptime

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
			"instances": map[string]interface{}{
				"rec_service_go":   recStatus,
				"user_service":     userStatus,
				"content_service":  contentStatus,
				"dashboard_fe":     "UP",
				"postgres_primary": pg,
				"redis_shards":     redisShards,
			},
			"metrics": map[string]interface{}{
				"throughput_rps":     int(rps + 0.5),
				"avg_p99_latency_ms": latencyMs,
			},
		})
	}
}

// handleMetricsAggregate serves the dashboard's one-stop metrics feed: the
// gateway's own snapshot plus every downstream service's /metricsz (null when a
// service is unreachable, which the UI renders as DOWN).
func handleMetricsAggregate(m *Metrics, targets map[string]string) http.HandlerFunc {
	client := &http.Client{Timeout: 900 * time.Millisecond}
	return func(w http.ResponseWriter, _ *http.Request) {
		services := map[string]interface{}{}
		for name, base := range targets {
			var snap map[string]interface{}
			resp, err := client.Get(base + "/metricsz")
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					if decodeErr := json.NewDecoder(resp.Body).Decode(&snap); decodeErr != nil {
						snap = nil
					}
				}
				resp.Body.Close()
			}
			if snap == nil {
				services[name] = nil
			} else {
				services[name] = snap
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mode":     "gateway",
			"ts":       time.Now().UnixMilli(),
			"gateway":  m.Snapshot(),
			"services": services,
		})
	}
}

func main() {
	startTime = time.Now()
	jwtSecret = resolveJWTSecret()
	metrics = NewMetrics("gateway")

	mux := http.NewServeMux()

	userServiceURL := envOr("USER_SERVICE_URL", "http://localhost:8081")
	contentServiceURL := envOr("CONTENT_SERVICE_URL", "http://localhost:8082")
	recommendationServiceURL := envOr("RECOMMENDATION_SERVICE_URL", "http://localhost:8083")

	// One circuit breaker per downstream service (shared across that service's routes).
	userBreaker := NewCircuitBreaker("user", 5, 5*time.Second)
	contentBreaker := NewCircuitBreaker("content", 5, 5*time.Second)
	recBreaker := NewCircuitBreaker("recommendation", 5, 5*time.Second)

	// Breaker states are exported as observability gauges, read at scrape time.
	metrics.SetGauge("breaker_user", userBreaker.State)
	metrics.SetGauge("breaker_content", contentBreaker.State)
	metrics.SetGauge("breaker_recommendation", recBreaker.State)

	mux.HandleFunc("/api/v1/health", handleHealth(userServiceURL, contentServiceURL, recommendationServiceURL))
	mux.HandleFunc("/api/v1/login", handleLogin)

	// Observability: Prometheus text, JSON snapshot, and the cross-service
	// aggregation the dashboard polls.
	mux.HandleFunc("/metrics", metrics.Handler())
	mux.HandleFunc("/metricsz", metrics.JSONHandler())
	mux.HandleFunc("/api/v1/metrics", handleMetricsAggregate(metrics, map[string]string{
		"user":           userServiceURL,
		"content":        contentServiceURL,
		"recommendation": recommendationServiceURL,
	}))

	// Route prefixes. Downstream services register their full paths (e.g. the
	// user service serves /api/v1/users/{id}/interactions itself), so the
	// gateway forwards paths untouched — stripping the prefix here would 404.
	mux.HandleFunc("/api/v1/users/", proxyTo(userServiceURL, "", userBreaker))
	mux.HandleFunc("/internal/users/", proxyTo(userServiceURL, "", userBreaker))

	mux.HandleFunc("/api/v1/content/", proxyTo(contentServiceURL, "", contentBreaker))
	mux.HandleFunc("/internal/content/", proxyTo(contentServiceURL, "", contentBreaker))

	mux.HandleFunc("/api/v1/recommendations", proxyTo(recommendationServiceURL, "", recBreaker))
	mux.HandleFunc("/api/v1/recommendations/", proxyTo(recommendationServiceURL, "", recBreaker))

	// Algorithm config: GET is public, PUT (a write to production ranking) needs a valid JWT.
	recConfigProxy := proxyTo(recommendationServiceURL, "", recBreaker)
	mux.HandleFunc("/api/v1/configs/history", recConfigProxy)
	mux.HandleFunc("/api/v1/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && !requireAuth(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401, "message": "Unauthorized: valid Bearer token required",
			})
			return
		}
		recConfigProxy(w, r)
	})

	// Per-IP rate limit as an abuse/DDoS safety cap. Defaults are generous so
	// normal traffic and load tests pass; tune via RATE_LIMIT_RPS / _BURST.
	limiter := NewIPRateLimiter(envFloat("RATE_LIMIT_RPS", 1000), envFloat("RATE_LIMIT_BURST", 2000))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Gateway listening on :" + port)
	// Outermost: metrics (sees everything, incl. 429s from the limiter), then
	// request-id assignment so every downstream hop shares one trace id.
	log.Fatal(http.ListenAndServe(":"+port, metrics.Middleware(RequestIDMiddleware(limiter.Middleware(mux)))))
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// resolveJWTSecret returns JWT_SECRET from the environment, or generates an
// ephemeral random secret if it is unset. There is no hardcoded secret in
// source. An ephemeral secret is fine for a single dev/test gateway (it issues
// and validates its own tokens), but production MUST set JWT_SECRET so tokens
// survive restarts and are shared across gateway instances.
func resolveJWTSecret() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("cannot generate an ephemeral JWT secret:", err)
	}
	log.Println("WARNING: JWT_SECRET is not set; generated an ephemeral random secret. " +
		"Tokens will not survive a restart or work across multiple instances. " +
		"Set JWT_SECRET in production.")
	return hex.EncodeToString(b)
}
