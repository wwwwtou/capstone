package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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

	// route prefixes
	mux.HandleFunc("/api/v1/users/", proxyTo("http://user:8081", "/api/v1/users"))
	mux.HandleFunc("/internal/users/", proxyTo("http://user:8081", "/internal/users"))

	mux.HandleFunc("/api/v1/content/", proxyTo("http://content:8082", "/api/v1/content"))
	mux.HandleFunc("/internal/content/", proxyTo("http://content:8082", "/internal/content"))

	mux.HandleFunc("/api/v1/recommendations", proxyTo("http://recommendation:8083", ""))
	mux.HandleFunc("/api/v1/recommendations/", proxyTo("http://recommendation:8083", ""))

	log.Println("Gateway listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
