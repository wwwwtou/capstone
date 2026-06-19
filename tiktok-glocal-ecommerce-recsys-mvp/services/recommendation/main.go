package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var cfgStore *ConfigStore

// Per-downstream circuit breakers guarding the synchronous calls the
// recommendation path makes to the user and content services.
var (
	userBreaker    = NewCircuitBreaker("user-service", 3, 5*time.Second)
	contentBreaker = NewCircuitBreaker("content-service", 3, 5*time.Second)
)

func main() {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres:5432/rec_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	cfgStore = &ConfigStore{DB: db}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"recommendation"}`))
	}).Methods("GET")
	r.HandleFunc("/api/v1/recommendations", handleRecommend).Methods("GET")
	r.HandleFunc("/api/v1/configs", handleGetConfig).Methods("GET")
	r.HandleFunc("/api/v1/configs", handleConfig).Methods("PUT")
	r.HandleFunc("/api/v1/configs/history", handleConfigHistory).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	addr := ":" + port
	log.Println("Recommendation service listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func handleRecommend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	// fetch user profile and content concurrently
	var wg sync.WaitGroup
	var profile UserProfile
	var videos []Video
	var pErr, cErr error
	wg.Add(2)

	go func() { defer wg.Done(); pErr = fetchProfile(ctx, userID, &profile) }()
	go func() { defer wg.Done(); cErr = fetchCandidates(ctx, &videos) }()
	wg.Wait()

	// Content candidates are essential: without them there is nothing to rank,
	// so a content outage (after retries + breaker) is a hard 503.
	if cErr != nil {
		log.Println("content service unavailable after retries/breaker:", cErr)
		http.Error(w, "content service unavailable", http.StatusServiceUnavailable)
		return
	}
	// The user profile is non-essential. On failure we degrade gracefully to a
	// cold-start (empty profile), which the EngagementStrategy serves as
	// globally-trending results, instead of failing the whole request.
	degraded := false
	if pErr != nil {
		log.Println("profile fetch failed, falling back to cold-start:", pErr)
		profile = UserProfile{UserID: userID}
		degraded = true
	}

	// read active strategy from rec_db
	strategyName := cfgStore.GetActiveStrategy()
	strat := StrategyFactory(strategyName)

	// Rank in-memory only
	ranked := strat.Rank(profile, videos)

	// Wrap in the envelope the dashboard/gateway expect.
	resp := map[string]interface{}{
		"trace_id": "req-" + time.Now().UTC().Format("20060102150405.000"),
		"code":     200,
		"message":  "success",
		"data": map[string]interface{}{
			"user_id":  userID,
			"strategy": strategyName,
			"degraded": degraded,
			"videos":   ranked,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleConfigHistory returns the recent deployment-log entries from the DB.
func handleConfigHistory(w http.ResponseWriter, r *http.Request) {
	history, err := cfgStore.GetHistory(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    history,
	})
}

// handleGetConfig returns the current active algorithm configuration.
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := cfgStore.GetActiveConfig()
	resp := map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"strategy_name": cfg.StrategyName,
			"weight":        cfg.Weight,
			"is_active":     true,
			"updated_at":    cfg.UpdatedAt,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// doGetJSON performs a single GET and decodes a 2xx JSON body into out. A
// transport error or non-2xx status is returned as an error so the circuit
// breaker counts it as a failure.
func doGetJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchProfile(ctx context.Context, userID string, out *UserProfile) error {
	userSvc := os.Getenv("USER_SERVICE_URL")
	if userSvc == "" {
		userSvc = "http://user:8081"
	}
	url := userSvc + "/internal/users/" + userID + "/profile"
	return callResilient(userBreaker, 3, 20*time.Millisecond, func() error {
		return doGetJSON(ctx, url, out)
	})
}

func fetchCandidates(ctx context.Context, out *[]Video) error {
	contentSvc := os.Getenv("CONTENT_SERVICE_URL")
	if contentSvc == "" {
		contentSvc = "http://content:8082"
	}
	url := contentSvc + "/internal/content/candidates"
	return callResilient(contentBreaker, 3, 20*time.Millisecond, func() error {
		return doGetJSON(ctx, url, out)
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		StrategyName string  `json:"strategy_name"`
		Weight       float64 `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.StrategyName == "" {
		http.Error(w, "strategy_name is required", http.StatusBadRequest)
		return
	}

	cfg, err := cfgStore.UpsertActiveConfig(payload.StrategyName, payload.Weight)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Record the change so the deployment log persists across navigation/restarts.
	if err := cfgStore.AddHistory(cfg.StrategyName, cfg.Weight); err != nil {
		log.Println("failed to append config history:", err)
	}

	resp := map[string]interface{}{
		"code":    200,
		"message": "Configuration deployed to Ranking Shards successfully",
		"data": map[string]interface{}{
			"strategy_name": cfg.StrategyName,
			"weight":        cfg.Weight,
			"is_active":     true,
			"updated_at":    cfg.UpdatedAt,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
