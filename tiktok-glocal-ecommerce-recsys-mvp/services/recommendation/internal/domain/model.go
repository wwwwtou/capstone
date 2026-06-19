// Package domain is the innermost layer of the recommendation service. It holds
// entities, value objects, the ranking strategy logic, and the repository
// *ports* (interfaces). It depends on nothing outside the standard library, so
// the core business rules stay isolated from DB, HTTP, and framework concerns.
package domain

import "time"

// UserProfile is the aggregated interest profile of a user.
type UserProfile struct {
	UserID string         `json:"user_id"`
	Tags   map[string]int `json:"tags"`
}

// Video is a candidate item plus the explainable ranking output.
type Video struct {
	VideoID   string    `json:"video_id"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	// Score and Reason are populated by the ranking strategy so the API can
	// explain *why* a video was ranked where it is.
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// ActiveConfig is the full algorithm configuration as seen by the dashboard.
// Name is the internal strategy class (e.g. EngagementStrategy) used by the
// StrategyFactory; StrategyName is the human-facing label (engagement /
// chronological / diversity).
type ActiveConfig struct {
	Name         string  `json:"name"`
	StrategyName string  `json:"strategy_name"`
	Weight       float64 `json:"weight"`
	UpdatedAt    string  `json:"updated_at"`
}

// ConfigChange is one persisted deployment-log entry.
type ConfigChange struct {
	StrategyName string  `json:"strategy_name"`
	Weight       float64 `json:"weight"`
	UpdatedAt    string  `json:"updated_at"`
}
