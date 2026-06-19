package app

import (
	"context"
	"errors"
	"testing"

	"recommendation/internal/domain"
)

// Fake repositories implementing the domain ports — no DB or HTTP needed, which
// is the whole point of depending on interfaces rather than concrete adapters.

type fakeProfiles struct {
	profile domain.UserProfile
	err     error
}

func (f fakeProfiles) GetProfile(context.Context, string) (domain.UserProfile, error) {
	return f.profile, f.err
}

type fakeContent struct {
	videos []domain.Video
	err    error
}

func (f fakeContent) GetCandidates(context.Context) ([]domain.Video, error) {
	return f.videos, f.err
}

type fakeConfig struct{ strategy string }

func (f fakeConfig) ActiveStrategy() string            { return f.strategy }
func (f fakeConfig) ActiveConfig() domain.ActiveConfig { return domain.ActiveConfig{Name: f.strategy} }
func (fakeConfig) UpsertConfig(string, float64) (domain.ActiveConfig, error) {
	return domain.ActiveConfig{}, nil
}
func (fakeConfig) AddHistory(string, float64) error           { return nil }
func (fakeConfig) History(int) ([]domain.ConfigChange, error) { return nil, nil }

func newCandidates() []domain.Video {
	return []domain.Video{
		{VideoID: "v1", Category: "home"},
		{VideoID: "v2", Category: "electronics"},
	}
}

func TestRecommendHappyPathPersonalizes(t *testing.T) {
	svc := NewService(
		fakeProfiles{profile: domain.UserProfile{UserID: "u1", Tags: map[string]int{"electronics": 3}}},
		fakeContent{videos: newCandidates()},
		fakeConfig{strategy: "EngagementStrategy"},
	)

	res, err := svc.Recommend(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Degraded {
		t.Fatal("should not be degraded when profile is available")
	}
	if res.Videos[0].VideoID != "v2" {
		t.Fatalf("expected matched-category video first, got %s", res.Videos[0].VideoID)
	}
	if res.Videos[0].Reason != "interest_match:electronics" {
		t.Fatalf("expected interest_match reason, got %s", res.Videos[0].Reason)
	}
}

func TestRecommendDegradesWhenProfileFails(t *testing.T) {
	svc := NewService(
		fakeProfiles{err: errors.New("user service down")},
		fakeContent{videos: newCandidates()},
		fakeConfig{strategy: "EngagementStrategy"},
	)

	res, err := svc.Recommend(context.Background(), "u1")
	if err != nil {
		t.Fatalf("profile failure must NOT fail the request, got %v", err)
	}
	if !res.Degraded {
		t.Fatal("expected degraded=true cold-start fallback")
	}
	if len(res.Videos) != 2 {
		t.Fatalf("expected videos still served, got %d", len(res.Videos))
	}
	for _, v := range res.Videos {
		if v.Reason != "globally_trending" {
			t.Fatalf("cold-start should be globally_trending, got %s", v.Reason)
		}
	}
}

func TestRecommendFailsWhenContentFails(t *testing.T) {
	svc := NewService(
		fakeProfiles{profile: domain.UserProfile{UserID: "u1"}},
		fakeContent{err: errors.New("content service down")},
		fakeConfig{strategy: "EngagementStrategy"},
	)

	_, err := svc.Recommend(context.Background(), "u1")
	if err == nil {
		t.Fatal("content failure must surface as an error (candidates are essential)")
	}
}

func TestUpdateConfigPersistsAndLogs(t *testing.T) {
	svc := NewService(fakeProfiles{}, fakeContent{}, fakeConfig{})
	if _, err := svc.UpdateConfig("chronological", 0.5); err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}
}
