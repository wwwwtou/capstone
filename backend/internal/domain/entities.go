package domain

import "context"

// VideoMetadata represents the core content entity
type VideoMetadata struct {
	ID       string            `json:"video_id"`
	Title    string            `json:"title"`
	AuthorID string            `json:"author_id"`
	Category string            `json:"category"`
	Tags     map[string]string `json:"tags"` // JSONB in DB
	Score    float64           `json:"score"`  // Calculated by Ranking Strategy
}

// AlgorithmConfig defines weights for the ranking engine
type AlgorithmConfig struct {
	ID           int     `json:"id"`
	StrategyName string  `json:"strategy_name"`
	Weight       float64 `json:"weight"`
	IsActive     bool    `json:"is_active"`
}

// UserProfile represents interest tags from Redis
type UserProfile struct {
	UserID    string             `json:"user_id"`
	Interests map[string]float64 `json:"interests"`
}

// Repositories (Domain Interfaces)
type VideoRepository interface {
	FetchCandidates(ctx context.Context, category string) ([]VideoMetadata, error)
}

type UserProfileRepository interface {
	GetProfile(ctx context.Context, userID string) (*UserProfile, error)
}

type ConfigRepository interface {
	GetActiveConfig(ctx context.Context) (*AlgorithmConfig, error)
}
