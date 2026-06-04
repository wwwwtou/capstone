package main

import (
	"testing"
	"time"
)

func TestEngagementStrategyRank_PrioritizesMatchingCategory(t *testing.T) {
	strategy := EngagementStrategy{}
	user := UserProfile{
		UserID: "u1",
		Tags: map[string]int{
			"electronics": 5,
			"fashion":     2,
		},
	}
	videos := []Video{
		{VideoID: "v1", Category: "fashion"},
		{VideoID: "v2", Category: "electronics"},
		{VideoID: "v3", Category: "home"},
	}

	ranked := strategy.Rank(user, videos)

	if len(ranked) != 3 {
		t.Fatalf("expected 3 videos, got %d", len(ranked))
	}
	if ranked[0].VideoID != "v2" {
		t.Fatalf("expected first video to be v2, got %s", ranked[0].VideoID)
	}
}

func TestChronologicalStrategyRank_NewestFirst(t *testing.T) {
	strategy := ChronologicalStrategy{}
	base := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	videos := []Video{
		{VideoID: "old", CreatedAt: base.Add(-2 * time.Hour)},
		{VideoID: "new", CreatedAt: base.Add(2 * time.Hour)},
		{VideoID: "mid", CreatedAt: base},
	}

	ranked := strategy.Rank(UserProfile{}, videos)

	if ranked[0].VideoID != "new" || ranked[1].VideoID != "mid" || ranked[2].VideoID != "old" {
		t.Fatalf("unexpected order: %s, %s, %s", ranked[0].VideoID, ranked[1].VideoID, ranked[2].VideoID)
	}
}

func TestStrategyFactory_ReturnsExpectedImplementation(t *testing.T) {
	if _, ok := StrategyFactory("EngagementStrategy").(EngagementStrategy); !ok {
		t.Fatalf("expected EngagementStrategy from factory")
	}
	if _, ok := StrategyFactory("ChronologicalStrategy").(ChronologicalStrategy); !ok {
		t.Fatalf("expected ChronologicalStrategy from factory")
	}
}

func TestStrategyFactory_DefaultsToEngagement(t *testing.T) {
	if _, ok := StrategyFactory("unknown").(EngagementStrategy); !ok {
		t.Fatalf("expected default strategy to be EngagementStrategy")
	}
}
