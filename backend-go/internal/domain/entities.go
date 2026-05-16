package domain

import "context"

// Video represents the video entity
type Video struct {
	ID    string  `json:"video_id"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
	Reason string  `json:"reason"`
}

// RecommendationResponse is the standard API response structure
type RecommendationResponse struct {
	TraceID string `json:"trace_id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Videos []Video `json:"videos"`
	} `json:"data"`
}

// UserProfile represents the user characteristics
type UserProfile struct {
	UserID string   `json:"user_id"`
	Tags   []string `json:"tags"`
}

// Repository Interfaces (Dependency Inversion)

type VideoRepository interface {
	GetCandidates(ctx context.Context) ([]Video, error)
}

type UserRepository interface {
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
}

type ConfigRepository interface {
	GetAlgorithmWeights(ctx context.Context) (map[string]float64, error)
}
