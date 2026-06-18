package main

import (
	"fmt"
	"sort"
)

type RankingStrategy interface {
	// Rank MUST be pure in-memory operation. No DB or network I/O here.
	Rank(user UserProfile, videos []Video) []Video
}

type EngagementStrategy struct{}

func (s EngagementStrategy) Rank(user UserProfile, videos []Video) []Video {
	// rank by simple tag match counts (higher match first)
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

type ChronologicalStrategy struct{}

func (s ChronologicalStrategy) Rank(user UserProfile, videos []Video) []Video {
	// newest first
	out := make([]Video, len(videos))
	copy(out, videos)
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	// Confidence decays with position so the freshest item ranks highest.
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
