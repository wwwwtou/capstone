package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Domain
type Video struct {
	ID    string  `json:"video_id"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
	Reason string  `json:"reason"`
}

type RecommendationResponse struct {
	TraceID string `json:"trace_id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Videos []Video `json:"videos"`
	} `json:"data"`
}

// Service Logic
func GetMockRecommendations(userID string) *RecommendationResponse {
	return &RecommendationResponse{
		TraceID: "req-" + time.Now().Format("20060102150405"),
		Code:    200,
		Message: "success",
		Data: struct {
			Videos []Video `json:"videos"`
		}{
			Videos: []Video{
				{ID: "v101", Title: "AI Revolution", Score: 0.98, Reason: "high_interest_tech"},
				{ID: "v202", Title: "Basketball Finals", Score: 0.85, Reason: "sports_preference"},
			},
		},
	}
}

func main() {
	http.HandleFunc("/api/v1/recommendations", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		res := GetMockRecommendations(userID)
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(res)
	})

	log.Println("Go Backend (Microservice 1: Rec-Service) listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
