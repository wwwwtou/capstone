package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// categoryFromMetadata is the core of profile aggregation: it decides which
// interaction events contribute a category tag to a user's interest profile.
func TestCategoryFromMetadata(t *testing.T) {
	cases := []struct {
		name    string
		meta    string
		wantCat string
		wantOK  bool
	}{
		{"valid category", `{"category":"tech"}`, "tech", true},
		{"extra fields ignored", `{"category":"food","score":9}`, "food", true},
		{"missing category", `{"author":"alice"}`, "", false},
		{"category not a string", `{"category":123}`, "", false},
		{"empty category string", `{"category":""}`, "", false},
		{"malformed json", `{not json`, "", false},
		{"empty blob", ``, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat, ok := categoryFromMetadata([]byte(tc.meta))
			if ok != tc.wantOK || cat != tc.wantCat {
				t.Fatalf("categoryFromMetadata(%q) = (%q, %v); want (%q, %v)",
					tc.meta, cat, ok, tc.wantCat, tc.wantOK)
			}
		})
	}
}

// A malformed interaction body must be rejected with 400 before any DB write,
// so the handler never reaches the (nil in this test) database.
func TestHandleInteractionRejectsMalformedBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/users/user_123/interactions", bytes.NewBufferString("{ this is not json"))
	rr := httptest.NewRecorder()

	handleInteraction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("malformed body: expected 400, got %d", rr.Code)
	}
}
