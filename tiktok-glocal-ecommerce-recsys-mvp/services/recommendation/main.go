// Command recommendation is the composition root: it wires the infrastructure
// adapters into the application service and exposes them over HTTP. All business
// logic lives in internal/{domain,app}; this file only assembles dependencies.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"recommendation/internal/app"
	"recommendation/internal/infra"
	"recommendation/internal/transport"
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

	userURL := envOr("USER_SERVICE_URL", "http://user:8081")
	contentURL := envOr("CONTENT_SERVICE_URL", "http://content:8082")

	// Infrastructure adapters implementing the domain ports.
	configRepo := infra.NewPostgresConfigRepository(db)
	profileRepo := infra.NewHTTPProfileRepository(userURL)
	contentRepo := infra.NewHTTPContentRepository(contentURL)

	// Application service + HTTP delivery.
	svc := app.NewService(profileRepo, contentRepo, configRepo)
	handler := transport.NewHandler(svc)

	r := mux.NewRouter()
	handler.Register(r)

	// Observability: metrics registry + breaker-state gauges + scrape endpoints.
	metrics := infra.NewMetrics("recommendation")
	metrics.SetGauge("breaker_user", profileRepo.BreakerState)
	metrics.SetGauge("breaker_content", contentRepo.BreakerState)
	r.HandleFunc("/metrics", metrics.Handler()).Methods(http.MethodGet)
	r.HandleFunc("/metricsz", metrics.JSONHandler()).Methods(http.MethodGet)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	addr := ":" + port
	log.Println("Recommendation service listening on", addr)
	log.Fatal(http.ListenAndServe(addr, metrics.Middleware(infra.RequestIDMiddleware(r))))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
