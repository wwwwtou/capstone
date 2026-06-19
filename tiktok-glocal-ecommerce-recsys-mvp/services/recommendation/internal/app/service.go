// Package app is the application layer: it orchestrates domain logic through the
// repository ports. It knows nothing about HTTP or SQL — only the domain
// interfaces — which keeps the use cases testable with fakes.
package app

import (
	"context"
	"log"
	"sync"

	"recommendation/internal/domain"
)

// Service implements the recommendation service's use cases.
type Service struct {
	profiles domain.ProfileRepository
	content  domain.ContentRepository
	configs  domain.ConfigRepository
}

func NewService(p domain.ProfileRepository, c domain.ContentRepository, cfg domain.ConfigRepository) *Service {
	return &Service{profiles: p, content: c, configs: cfg}
}

// Result is the output of the Recommend use case.
type Result struct {
	UserID   string
	Strategy string
	Degraded bool
	Videos   []domain.Video
}

// Recommend fetches the profile and candidates concurrently, applies graceful
// degradation (a missing profile falls back to a cold-start empty profile), and
// ranks with the active strategy. Candidates are essential: a content failure
// is returned as an error.
func (s *Service) Recommend(ctx context.Context, userID string) (Result, error) {
	var (
		wg         sync.WaitGroup
		profile    domain.UserProfile
		videos     []domain.Video
		pErr, cErr error
	)
	wg.Add(2)
	go func() { defer wg.Done(); profile, pErr = s.profiles.GetProfile(ctx, userID) }()
	go func() { defer wg.Done(); videos, cErr = s.content.GetCandidates(ctx) }()
	wg.Wait()

	if cErr != nil {
		return Result{}, cErr // content is essential
	}

	degraded := false
	if pErr != nil {
		// Non-essential: cold-start fallback keeps the request serving results.
		log.Println("profile fetch failed, falling back to cold-start:", pErr)
		profile = domain.UserProfile{UserID: userID}
		degraded = true
	}

	strategyName := s.configs.ActiveStrategy()
	ranked := domain.StrategyFactory(strategyName).Rank(profile, videos)

	return Result{UserID: userID, Strategy: strategyName, Degraded: degraded, Videos: ranked}, nil
}

// GetConfig returns the active algorithm configuration.
func (s *Service) GetConfig() domain.ActiveConfig {
	return s.configs.ActiveConfig()
}

// UpdateConfig persists a new active configuration and appends a deployment-log
// entry so the change survives navigation/restarts.
func (s *Service) UpdateConfig(strategyName string, weight float64) (domain.ActiveConfig, error) {
	cfg, err := s.configs.UpsertConfig(strategyName, weight)
	if err != nil {
		return domain.ActiveConfig{}, err
	}
	if err := s.configs.AddHistory(cfg.StrategyName, cfg.Weight); err != nil {
		log.Println("failed to append config history:", err)
	}
	return cfg, nil
}

// History returns recent deployment-log entries, newest first.
func (s *Service) History(limit int) ([]domain.ConfigChange, error) {
	return s.configs.History(limit)
}
