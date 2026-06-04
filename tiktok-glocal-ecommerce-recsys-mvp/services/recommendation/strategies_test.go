package main

import (
	"testing"
	"time"
)

func TestEngagementStrategyRank(t *testing.T) {
	tests := []struct {
		name     string
		user     UserProfile
		videos   []Video
		wantTop  string
		wantSize int
	}{
		{
			name: "prioritizes_matching_category",
			user: UserProfile{UserID: "u1", Tags: map[string]int{"electronics": 5, "fashion": 2}},
			videos: []Video{
				{VideoID: "v1", Category: "fashion"},
				{VideoID: "v2", Category: "electronics"},
				{VideoID: "v3", Category: "home"},
			},
			wantTop:  "v2",
			wantSize: 3,
		},
		{
			name:     "handles_nil_tags",
			user:     UserProfile{UserID: "u2"},
			videos:   []Video{{VideoID: "v1", Category: "fashion"}, {VideoID: "v2", Category: "electronics"}},
			wantTop:  "v1",
			wantSize: 2,
		},
		{
			name:     "returns_empty_for_empty_list",
			user:     UserProfile{UserID: "u3", Tags: map[string]int{"fashion": 1}},
			videos:   nil,
			wantTop:  "",
			wantSize: 0,
		},
		{
			name: "preserves_stable_order_on_ties",
			user: UserProfile{UserID: "u4", Tags: map[string]int{"fashion": 1}},
			videos: []Video{
				{VideoID: "v1", Category: "home"},
				{VideoID: "v2", Category: "home"},
			},
			wantTop:  "v1",
			wantSize: 2,
		},
		{
			name: "prioritizes_higher_tag_weight_over_lower_weight_match",
			user: UserProfile{UserID: "u5", Tags: map[string]int{"home": 1, "fashion": 9}},
			videos: []Video{
				{VideoID: "v1", Category: "home"},
				{VideoID: "v2", Category: "fashion"},
			},
			wantTop:  "v2",
			wantSize: 2,
		},
		{
			name: "keeps_original_order_when_all_scores_equal",
			user: UserProfile{UserID: "u6", Tags: map[string]int{"sports": 1}},
			videos: []Video{
				{VideoID: "v1", Category: "finance"},
				{VideoID: "v2", Category: "travel"},
				{VideoID: "v3", Category: "music"},
			},
			wantTop:  "v1",
			wantSize: 3,
		},
		{
			name: "handles_multiple_matching_videos_in_rank_order",
			user: UserProfile{UserID: "u7", Tags: map[string]int{"electronics": 3, "fashion": 1}},
			videos: []Video{
				{VideoID: "v1", Category: "fashion"},
				{VideoID: "v2", Category: "electronics"},
				{VideoID: "v3", Category: "electronics"},
			},
			wantTop:  "v2",
			wantSize: 3,
		},
		{
			name: "supports_multiple_distinct_tags",
			user: UserProfile{UserID: "u8", Tags: map[string]int{"music": 4, "travel": 2}},
			videos: []Video{
				{VideoID: "v1", Category: "travel"},
				{VideoID: "v2", Category: "music"},
				{VideoID: "v3", Category: "sports"},
			},
			wantTop:  "v2",
			wantSize: 3,
		},
		{
			name: "handles_many_videos_without_scores",
			user: UserProfile{UserID: "u9"},
			videos: []Video{
				{VideoID: "v1", Category: "sports"},
				{VideoID: "v2", Category: "news"},
				{VideoID: "v3", Category: "tech"},
				{VideoID: "v4", Category: "gaming"},
			},
			wantTop:  "v1",
			wantSize: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ranked := EngagementStrategy{}.Rank(tc.user, tc.videos)
			if len(ranked) != tc.wantSize {
				t.Fatalf("expected %d videos, got %d", tc.wantSize, len(ranked))
			}
			if tc.wantSize > 0 && ranked[0].VideoID != tc.wantTop {
				t.Fatalf("expected first video to be %s, got %s", tc.wantTop, ranked[0].VideoID)
			}
		})
	}
}

func TestChronologicalStrategyRank(t *testing.T) {
	base := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		videos    []Video
		wantOrder []string
	}{
		{
			name: "newest_first",
			videos: []Video{
				{VideoID: "old", CreatedAt: base.Add(-2 * time.Hour)},
				{VideoID: "new", CreatedAt: base.Add(2 * time.Hour)},
				{VideoID: "mid", CreatedAt: base},
			},
			wantOrder: []string{"new", "mid", "old"},
		},
		{
			name:      "handles_single_item",
			videos:    []Video{{VideoID: "only", CreatedAt: base}},
			wantOrder: []string{"only"},
		},
		{
			name:      "empty_list_returns_empty",
			videos:    nil,
			wantOrder: []string{},
		},
		{
			name: "handles_equal_timestamps_stably",
			videos: []Video{
				{VideoID: "a", CreatedAt: base},
				{VideoID: "b", CreatedAt: base},
			},
			wantOrder: []string{"a", "b"},
		},
		{
			name: "keeps_descending_order_for_three_items",
			videos: []Video{
				{VideoID: "older", CreatedAt: base.Add(-4 * time.Hour)},
				{VideoID: "newer", CreatedAt: base.Add(4 * time.Hour)},
				{VideoID: "middle", CreatedAt: base.Add(-1 * time.Hour)},
			},
			wantOrder: []string{"newer", "middle", "older"},
		},
		{
			name: "handles_far_apart_timestamps",
			videos: []Video{
				{VideoID: "last", CreatedAt: base.Add(-72 * time.Hour)},
				{VideoID: "first", CreatedAt: base.Add(72 * time.Hour)},
			},
			wantOrder: []string{"first", "last"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ranked := ChronologicalStrategy{}.Rank(UserProfile{}, tc.videos)
			if len(ranked) != len(tc.wantOrder) {
				t.Fatalf("expected %d videos, got %d", len(tc.wantOrder), len(ranked))
			}
			for i, wantID := range tc.wantOrder {
				if ranked[i].VideoID != wantID {
					t.Fatalf("expected position %d to be %s, got %s", i, wantID, ranked[i].VideoID)
				}
			}
		})
	}
}

func TestStrategyFactory(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "returns_engagement", in: "EngagementStrategy", want: "engagement"},
		{name: "returns_chronological", in: "ChronologicalStrategy", want: "chronological"},
		{name: "defaults_unknown_to_engagement", in: "unknown", want: "engagement"},
		{name: "defaults_empty_string_to_engagement", in: "", want: "engagement"},
		{name: "defaults_whitespace_to_engagement", in: "   ", want: "engagement"},
		{name: "defaults_case_mismatch_to_engagement", in: "engagementstrategy", want: "engagement"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			switch StrategyFactory(tc.in).(type) {
			case EngagementStrategy:
				if tc.want != "engagement" {
					t.Fatalf("expected %s, got engagement", tc.want)
				}
			case ChronologicalStrategy:
				if tc.want != "chronological" {
					t.Fatalf("expected %s, got chronological", tc.want)
				}
			default:
				t.Fatalf("unexpected strategy type for %s", tc.in)
			}
		})
	}
}
