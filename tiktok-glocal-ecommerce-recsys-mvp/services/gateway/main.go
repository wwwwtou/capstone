package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func proxyTo(target string, removePrefix string) http.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)
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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8081"
	}
	contentServiceURL := os.Getenv("CONTENT_SERVICE_URL")
	if contentServiceURL == "" {
		contentServiceURL = "http://localhost:8082"
	}
	recommendationServiceURL := os.Getenv("RECOMMENDATION_SERVICE_URL")
	if recommendationServiceURL == "" {
		recommendationServiceURL = "http://localhost:8083"
	}

	// route prefixes
	mux.HandleFunc("/api/v1/users/", proxyTo(userServiceURL, "/api/v1/users"))
	mux.HandleFunc("/internal/users/", proxyTo(userServiceURL, "/internal/users"))

	mux.HandleFunc("/api/v1/content/", proxyTo(contentServiceURL, "/api/v1/content"))
	mux.HandleFunc("/internal/content/", proxyTo(contentServiceURL, "/internal/content"))

	mux.HandleFunc("/api/v1/recommendations", proxyTo(recommendationServiceURL, ""))
	mux.HandleFunc("/api/v1/recommendations/", proxyTo(recommendationServiceURL, ""))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Gateway listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
