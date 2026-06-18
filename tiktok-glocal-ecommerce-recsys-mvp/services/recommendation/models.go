package main

import "time"

type UserProfile struct {
	UserID string         `json:"user_id"`
	Tags   map[string]int `json:"tags"`
}

type Video struct {
	VideoID   string    `json:"video_id"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	// Score and Reason are populated by the ranking strategy so the
	// API can explain *why* a video was ranked where it is.
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}
