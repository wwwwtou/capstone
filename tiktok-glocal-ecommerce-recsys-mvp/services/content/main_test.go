package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("health: expected application/json, got %q", ct)
	}
	var body struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("health: body is not valid json: %v", err)
	}
	if body.Status != "ok" || body.Service != "content" {
		t.Fatalf("health: unexpected body %+v", body)
	}
}

// The candidates endpoint serializes Video rows straight to the gateway, so the
// JSON field names are an API contract the recommendation service depends on.
func TestVideoJSONContract(t *testing.T) {
	v := Video{
		VideoID:   "v1",
		Author:    "alice",
		Category:  "tech",
		Title:     "hello world",
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal Video: %v", err)
	}
	out := string(b)
	for _, field := range []string{`"video_id"`, `"author"`, `"category"`, `"title"`, `"created_at"`} {
		if !strings.Contains(out, field) {
			t.Fatalf("Video JSON missing field %s; got %s", field, out)
		}
	}
}
