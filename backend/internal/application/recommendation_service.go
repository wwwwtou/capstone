package application

import (
	"context"
	"fmt"
	"sync"
	"backend/internal/domain"
)

type RecommendationService struct {
	videoRepo  domain.VideoRepository
	userRepo   domain.UserProfileRepository
	configRepo domain.ConfigRepository
}

func NewRecommendationService(vr domain.VideoRepository, ur domain.UserProfileRepository, cr domain.ConfigRepository) *RecommendationService {
	return &RecommendationService{vr, ur, cr}
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, userID string) ([]domain.VideoMetadata, error) {
	var (
		wg      sync.WaitGroup
		errs    = make(chan error, 2)
		profile *domain.UserProfile
		videos  []domain.VideoMetadata
		config  *domain.AlgorithmConfig
	)

	// Step 1: Concurrent Data Fetching (Goroutines & WaitGroup)
	// This demonstrates performance optimization for MTech defense
	wg.Add(2)
	
	go func() {
		defer wg.Done()
		p, err := s.userRepo.GetProfile(ctx, userID)
		if err != nil {
			errs <- fmt.Errorf("redis_error: %v", err)
			return
		}
		profile = p
	}()

	go func() {
		defer wg.Done()
		// Fetching all for simplicity, or filtered by global hot category
		v, err := s.videoRepo.FetchCandidates(ctx, "all")
		if err != nil {
			errs <- fmt.Errorf("postgres_error: %v", err)
			return
		}
		videos = v
	}()

	wg.Wait()
	close(errs)

	// Check if any error occurred during concurrent fetching
	for err := range errs {
		return nil, err
	}

	// Step 2: Fetch Active Config
	config, err := s.configRepo.GetActiveConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Apply Strategy Pattern
	var strategy RankingStrategy
	switch config.StrategyName {
	case "engagement":
		strategy = &EngagementStrategy{}
	case "chronological":
		strategy = &ChronologicalStrategy{}
	default:
		strategy = &EngagementStrategy{}
	}

	ranked := strategy.Rank(videos, profile, config.Weight)
	return ranked, nil
}
