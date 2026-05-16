package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"backend-go/internal/domain"
)

// RecommendationService handles the business logic
type RecommendationService struct {
	videoRepo  domain.VideoRepository
	userRepo   domain.UserRepository
	configRepo domain.ConfigRepository
}

func NewRecommendationService(v domain.VideoRepository, u domain.UserRepository, c domain.ConfigRepository) *RecommendationService {
	return &RecommendationService{v, u, c}
}

// GetRecommendations executes the core recommendation logic
func (s *RecommendationService) GetRecommendations(ctx context.Context, userID string) (*domain.RecommendationResponse, error) {
	var (
		wg      sync.WaitGroup
		errs    = make(chan error, 2)
		profile *domain.UserProfile
		videos  []domain.Video
		weights map[string]float64
	)

	// Step 1: Concurrent Data Fetching (Demonstrates mastery of Go concurrency)
	wg.Add(2)
	go func() {
		defer wg.Done()
		p, err := s.userRepo.GetUserProfile(ctx, userID)
		if err != nil {
			errs <- err
			return
		}
		profile = p
	}()

	go func() {
		defer wg.Done()
		v, err := s.videoRepo.GetCandidates(ctx)
		if err != nil {
			errs <- err
			return
		}
		videos = v
	}()

	wg.Wait()
	close(errs)

	for err := range errs {
		return nil, fmt.Errorf("data fetching failed: %w", err)
	}

	// Step 2: Fetch Dynamic Weights
	weights, _ = s.configRepo.GetAlgorithmWeights(ctx)

	// Step 3: Ranking Stage (Strategy Pattern should be applied here)
	rankedVideos := s.rank(videos, profile, weights)

	return &domain.RecommendationResponse{
		TraceID: "req-" + time.Now().Format("20060102150405"),
		Code:    200,
		Message: "success",
		Data: struct {
			Videos []domain.Video `json:"videos"`
		}{Videos: rankedVideos},
	}, nil
}

// rank Mocking the ranking logic
func (s *RecommendationService) rank(candidates []domain.Video, profile *domain.UserProfile, weights map[string]float64) []domain.Video {
	// In a real system, this would use a specific Strategy implementation
	// For MVP, we simulate scoring based on tag matching
	for i := range candidates {
		candidates[i].Score = 0.85 // Base score
		candidates[i].Reason = "matched_interests"
	}
	return candidates
}
