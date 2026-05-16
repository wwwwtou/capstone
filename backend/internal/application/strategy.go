package application

import (
	"sort"
	"backend/internal/domain"
)

// RankingStrategy defines the interface for Strategy Pattern
type RankingStrategy interface {
	Rank(videos []domain.VideoMetadata, profile *domain.UserProfile, globalWeight float64) []domain.VideoMetadata
}

// EngagementStrategy ranks based on interest matching intensity
type EngagementStrategy struct{}

func (s *EngagementStrategy) Rank(videos []domain.VideoMetadata, profile *domain.UserProfile, weight float64) []domain.VideoMetadata {
	for i := range videos {
		score := 0.5 // Base
		// Simulate tag matching logic
		if val, ok := profile.Interests[videos[i].Category]; ok {
			score += val * weight
		}
		videos[i].Score = score
	}
	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Score > videos[j].Score
	})
	return videos
}

// ChronologicalStrategy ranks by recency (Mock)
type ChronologicalStrategy struct{}

func (s *ChronologicalStrategy) Rank(videos []domain.VideoMetadata, profile *domain.UserProfile, weight float64) []domain.VideoMetadata {
	// In reality, this would sort by a timestamp. Mocking for MVP.
	for i := range videos {
		videos[i].Score = 0.9 - (float64(i) * 0.01)
	}
	return videos
}
