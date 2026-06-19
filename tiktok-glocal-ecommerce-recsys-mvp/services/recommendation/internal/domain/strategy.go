package domain

import (
	"fmt"
	"sort"
)

// RankingStrategy is the Strategy pattern abstraction for ordering candidate
// videos. Implementations MUST be pure in-memory operations (no DB / network).
type RankingStrategy interface {
	Rank(user UserProfile, videos []Video) []Video
}

// EngagementStrategy ranks by how well a video's category matches the user's
// interest tags, falling back to globally-trending order.
type EngagementStrategy struct{}

func (EngagementStrategy) Rank(user UserProfile, videos []Video) []Video {
	scored := make([]struct {
		v     Video
		score int
	}, 0, len(videos))
	for _, v := range videos {
		score := 0
		if user.Tags != nil {
			if c, ok := user.Tags[v.Category]; ok {
				score += c
			}
		}
		// Translate the integer match count into a display confidence in
		// [0.60, 0.99] and an explainable reason string.
		conf := 0.60 + 0.08*float64(score)
		if conf > 0.99 {
			conf = 0.99
		}
		v.Score = conf
		if score > 0 {
			v.Reason = fmt.Sprintf("interest_match:%s", v.Category)
		} else {
			v.Reason = "globally_trending"
		}
		scored = append(scored, struct {
			v     Video
			score int
		}{v: v, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	out := make([]Video, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.v)
	}
	return out
}

// ChronologicalStrategy ranks newest-first.
type ChronologicalStrategy struct{}

func (ChronologicalStrategy) Rank(_ UserProfile, videos []Video) []Video {
	out := make([]Video, len(videos))
	copy(out, videos)
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	for i := range out {
		conf := 0.95 - 0.05*float64(i)
		if conf < 0.30 {
			conf = 0.30
		}
		out[i].Score = conf
		out[i].Reason = "recency"
	}
	return out
}

// StrategyFactory returns the ranking strategy for an internal strategy class.
func StrategyFactory(name string) RankingStrategy {
	switch name {
	case "EngagementStrategy":
		return EngagementStrategy{}
	case "ChronologicalStrategy":
		return ChronologicalStrategy{}
	default:
		return EngagementStrategy{}
	}
}

// StrategyClassFor maps a UI label to the internal strategy class name.
func StrategyClassFor(strategyName string) string {
	switch strategyName {
	case "chronological":
		return "ChronologicalStrategy"
	case "engagement", "diversity":
		return "EngagementStrategy"
	default:
		return "EngagementStrategy"
	}
}
