package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres:5432/content_db?sslmode=disable"
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)

	r := mux.NewRouter()
	r.HandleFunc("/healthz", healthHandler).Methods("GET")
	r.HandleFunc("/internal/content/candidates", handleCandidates).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	addr := ":" + port
	log.Println("Content service listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"content"}`))
}

type Video struct {
	VideoID   string    `json:"video_id"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func handleCandidates(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT video_id, author, category, title, created_at FROM videos ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.VideoID, &v.Author, &v.Category, &v.Title, &v.CreatedAt); err == nil {
			out = append(out, v)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
