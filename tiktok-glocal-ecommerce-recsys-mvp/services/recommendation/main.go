package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var cfgStore *ConfigStore

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
	r.HandleFunc("/api/v1/recommendations", handleRecommend).Methods("GET")
	r.HandleFunc("/api/v1/configs", handleConfig).Methods("PUT")

	addr := ":8083"
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
	if pErr != nil || cErr != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	// read active strategy from rec_db
	strategyName := cfgStore.GetActiveStrategy()
	strat := StrategyFactory(strategyName)

	// Rank in-memory only
	ranked := strat.Rank(profile, videos)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ranked)
}

func fetchProfile(ctx context.Context, userID string, out *UserProfile) error {
	userSvc := os.Getenv("USER_SERVICE_URL")
	if userSvc == "" {
		userSvc = "http://user:8081"
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", userSvc+"/internal/users/"+userID+"/profile", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchCandidates(ctx context.Context, out *[]Video) error {
	contentSvc := os.Getenv("CONTENT_SERVICE_URL")
	if contentSvc == "" {
		contentSvc = "http://content:8082"
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", contentSvc+"/internal/content/candidates", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// upsert into configs
	raw, _ := json.Marshal(payload)
	_, err := cfgStore.DB.Exec("INSERT INTO configs (key, value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=$2", "active_strategy", raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
