package domain

import "context"

// The repository *ports* below are the boundaries the application layer depends
// on. Infrastructure adapters (HTTP clients, Postgres) implement them, so the
// dependency arrows point inward (infra -> domain), per clean architecture.

// ProfileRepository fetches a user's interest profile (from the user service).
type ProfileRepository interface {
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
}

// ContentRepository fetches candidate videos (from the content service).
type ContentRepository interface {
	GetCandidates(ctx context.Context) ([]Video, error)
}

// ConfigRepository reads and writes the active ranking configuration and its
// deployment-log history (in rec_db).
type ConfigRepository interface {
	ActiveStrategy() string
	ActiveConfig() ActiveConfig
	UpsertConfig(strategyName string, weight float64) (ActiveConfig, error)
	AddHistory(strategyName string, weight float64) error
	History(limit int) ([]ConfigChange, error)
}
