package main

import "testing"

func TestStrategyClassFor(t *testing.T) {
	cases := map[string]string{
		"engagement":    "EngagementStrategy",
		"chronological": "ChronologicalStrategy",
		"diversity":     "EngagementStrategy", // no diversity impl yet; falls back
		"unknown":       "EngagementStrategy",
		"":              "EngagementStrategy",
	}
	for in, want := range cases {
		if got := strategyClassFor(in); got != want {
			t.Fatalf("strategyClassFor(%q) = %q, want %q", in, got, want)
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
	// The matched-category video should rank first and be tagged as a match.
	if ranked[0].VideoID != "v1" {
		t.Fatalf("expected matched-category video first, got %s", ranked[0].VideoID)
	}
	if ranked[0].Reason != "interest_match:electronics" {
		t.Fatalf("unexpected reason for matched video: %s", ranked[0].Reason)
	}
}
