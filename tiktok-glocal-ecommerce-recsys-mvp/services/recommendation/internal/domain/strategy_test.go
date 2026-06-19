package domain

import (
	"testing"
	"time"
)

func TestStrategyClassFor(t *testing.T) {
	cases := map[string]string{
		"engagement":    "EngagementStrategy",
		"chronological": "ChronologicalStrategy",
		"diversity":     "EngagementStrategy", // no diversity impl yet; falls back
		"unknown":       "EngagementStrategy",
		"":              "EngagementStrategy",
	}
	for in, want := range cases {
		if got := StrategyClassFor(in); got != want {
			t.Fatalf("StrategyClassFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEngagementStrategySetsScoreAndReason(t *testing.T) {
	user := UserProfile{UserID: "u1", Tags: map[string]int{"electronics": 3}}
	videos := []Video{
		{VideoID: "v1", Category: "electronics"},
		{VideoID: "v2", Category: "home"},
	}
	ranked := EngagementStrategy{}.Rank(user, videos)

	for _, v := range ranked {
		if v.Score <= 0 {
			t.Fatalf("video %s should have a positive score, got %v", v.VideoID, v.Score)
		}
		if v.Reason == "" {
			t.Fatalf("video %s should have a non-empty reason", v.VideoID)
		}
	}
	if ranked[0].VideoID != "v1" {
		t.Fatalf("expected matched-category video first, got %s", ranked[0].VideoID)
	}
	if ranked[0].Reason != "interest_match:electronics" {
		t.Fatalf("unexpected reason for matched video: %s", ranked[0].Reason)
	}
}

func TestEngagementStrategyRank(t *testing.T) {
	cases := []struct {
		name      string
		user      UserProfile
		videos    []Video
		wantFirst string
		wantLen   int
	}{
		{"prioritizes matching category",
			UserProfile{Tags: map[string]int{"tech": 5}},
			[]Video{{VideoID: "a", Category: "home"}, {VideoID: "b", Category: "tech"}}, "b", 2},
		{"handles nil tags",
			UserProfile{},
			[]Video{{VideoID: "a", Category: "home"}, {VideoID: "b", Category: "tech"}}, "a", 2},
		{"returns empty for empty list",
			UserProfile{Tags: map[string]int{"tech": 1}}, nil, "", 0},
		{"preserves stable order on ties",
			UserProfile{Tags: map[string]int{}},
			[]Video{{VideoID: "a"}, {VideoID: "b"}, {VideoID: "c"}}, "a", 3},
		{"prioritizes higher tag weight over lower weight match",
			UserProfile{Tags: map[string]int{"tech": 1, "food": 9}},
			[]Video{{VideoID: "a", Category: "tech"}, {VideoID: "b", Category: "food"}}, "b", 2},
		{"keeps original order when all scores equal",
			UserProfile{Tags: map[string]int{"x": 2}},
			[]Video{{VideoID: "a", Category: "y"}, {VideoID: "b", Category: "z"}}, "a", 2},
		{"handles multiple matching videos in rank order",
			UserProfile{Tags: map[string]int{"tech": 4, "food": 2}},
			[]Video{{VideoID: "a", Category: "food"}, {VideoID: "b", Category: "tech"}, {VideoID: "c", Category: "home"}}, "b", 3},
		{"supports multiple distinct tags",
			UserProfile{Tags: map[string]int{"a": 1, "b": 1, "c": 1}},
			[]Video{{VideoID: "v", Category: "a"}, {VideoID: "w", Category: "d"}}, "v", 2},
		{"handles many videos without scores",
			UserProfile{Tags: map[string]int{"none": 5}},
			[]Video{{VideoID: "a"}, {VideoID: "b"}, {VideoID: "c"}, {VideoID: "d"}}, "a", 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EngagementStrategy{}.Rank(tc.user, tc.videos)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 && got[0].VideoID != tc.wantFirst {
				t.Fatalf("first = %s, want %s", got[0].VideoID, tc.wantFirst)
			}
		})
	}
}

func TestChronologicalStrategyRank(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		videos    []Video
		wantFirst string
		wantLen   int
	}{
		{"newest first",
			[]Video{
				{VideoID: "old", CreatedAt: now.Add(-2 * time.Hour)},
				{VideoID: "new", CreatedAt: now},
			}, "new", 2},
		{"handles single item", []Video{{VideoID: "only", CreatedAt: now}}, "only", 1},
		{"empty list returns empty", nil, "", 0},
		{"handles equal timestamps stably",
			[]Video{
				{VideoID: "a", CreatedAt: now},
				{VideoID: "b", CreatedAt: now},
			}, "a", 2},
		{"keeps descending order for three items",
			[]Video{
				{VideoID: "b", CreatedAt: now.Add(-1 * time.Hour)},
				{VideoID: "c", CreatedAt: now.Add(-2 * time.Hour)},
				{VideoID: "a", CreatedAt: now},
			}, "a", 3},
		{"handles far apart timestamps",
			[]Video{
				{VideoID: "ancient", CreatedAt: now.Add(-9000 * time.Hour)},
				{VideoID: "fresh", CreatedAt: now},
			}, "fresh", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ChronologicalStrategy{}.Rank(UserProfile{}, tc.videos)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			if tc.wantLen > 0 {
				if got[0].VideoID != tc.wantFirst {
					t.Fatalf("first = %s, want %s", got[0].VideoID, tc.wantFirst)
				}
				if got[0].Reason != "recency" {
					t.Fatalf("reason = %s, want recency", got[0].Reason)
				}
			}
		})
	}
}

func TestStrategyFactory(t *testing.T) {
	cases := []struct {
		name            string
		input           string
		wantChronologic bool
	}{
		{"returns engagement", "EngagementStrategy", false},
		{"returns chronological", "ChronologicalStrategy", true},
		{"defaults unknown to engagement", "SomethingElse", false},
		{"defaults empty string to engagement", "", false},
		{"defaults whitespace to engagement", "   ", false},
		{"defaults case mismatch to engagement", "engagementstrategy", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StrategyFactory(tc.input)
			if tc.wantChronologic {
				if _, ok := got.(ChronologicalStrategy); !ok {
					t.Fatalf("StrategyFactory(%q) did not return ChronologicalStrategy", tc.input)
				}
				return
			}
			if _, ok := got.(EngagementStrategy); !ok {
				t.Fatalf("StrategyFactory(%q) should default to EngagementStrategy", tc.input)
			}
		})
	}
}
