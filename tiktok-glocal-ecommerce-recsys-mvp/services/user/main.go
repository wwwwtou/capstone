package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB
var rdb *redis.Client

func main() {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres:5432/user_db?sslmode=disable"
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatal(err)
		}
		rdb = redis.NewClient(opt)
	} else {
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}
		rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"user"}`))
	}).Methods("GET")
	r.HandleFunc("/api/v1/users/{id}/interactions", handleInteraction).Methods("POST")
	r.HandleFunc("/internal/users/{id}/profile", handleProfile).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	addr := ":" + port
	log.Println("User service listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

type Interaction struct {
	EventType string                 `json:"event_type"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func handleInteraction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	var it Interaction
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	meta, _ := json.Marshal(it.Metadata)
	_, err := db.ExecContext(ctx, "INSERT INTO interactions (user_id, event_type, metadata) VALUES ($1,$2,$3)", id, it.EventType, meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// simple cache invalidation
	rdb.Del(ctx, fmt.Sprintf("profile:%s", id))
	w.WriteHeader(http.StatusNoContent)
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	key := fmt.Sprintf("profile:%s", id)
	cached, err := rdb.Get(ctx, key).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}
	// build a simple profile from interactions
	rows, err := db.QueryContext(ctx, "SELECT event_type, metadata FROM interactions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	tags := map[string]int{}
	for rows.Next() {
		var event string
		var meta []byte
		if err := rows.Scan(&event, &meta); err != nil {
			continue
		}
		var m map[string]interface{}
		_ = json.Unmarshal(meta, &m)
		if cat, ok := m["category"].(string); ok {
			tags[cat]++
		}
	}
	// build top tags
	profile := map[string]interface{}{"user_id": id, "tags": tags}
	b, _ := json.Marshal(profile)
	rdb.Set(ctx, key, string(b), time.Minute*10)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
