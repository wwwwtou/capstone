package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"recommendation/internal/domain"
)

// doGetJSON performs a single GET and decodes a 2xx JSON body into out. A
// transport error or non-2xx status is returned as an error so the circuit
// breaker counts it as a failure.
func doGetJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// HTTPProfileRepository implements domain.ProfileRepository against the user
// service, guarded by a circuit breaker + retry.
type HTTPProfileRepository struct {
	baseURL string
	breaker *CircuitBreaker
}

func NewHTTPProfileRepository(baseURL string) *HTTPProfileRepository {
	return &HTTPProfileRepository{baseURL: baseURL, breaker: NewCircuitBreaker("user-service", 3, 5*time.Second)}
}

func (r *HTTPProfileRepository) GetProfile(ctx context.Context, userID string) (domain.UserProfile, error) {
	var profile domain.UserProfile
	url := r.baseURL + "/internal/users/" + userID + "/profile"
	err := callResilient(r.breaker, 3, 20*time.Millisecond, func() error {
		return doGetJSON(ctx, url, &profile)
	})
	return profile, err
}

// HTTPContentRepository implements domain.ContentRepository against the content
// service, guarded by a circuit breaker + retry.
type HTTPContentRepository struct {
	baseURL string
	breaker *CircuitBreaker
}

func NewHTTPContentRepository(baseURL string) *HTTPContentRepository {
	return &HTTPContentRepository{baseURL: baseURL, breaker: NewCircuitBreaker("content-service", 3, 5*time.Second)}
}

func (r *HTTPContentRepository) GetCandidates(ctx context.Context) ([]domain.Video, error) {
	var videos []domain.Video
	url := r.baseURL + "/internal/content/candidates"
	err := callResilient(r.breaker, 3, 20*time.Millisecond, func() error {
		return doGetJSON(ctx, url, &videos)
	})
	return videos, err
}
