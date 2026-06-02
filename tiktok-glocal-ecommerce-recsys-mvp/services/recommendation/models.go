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
}
