package main

import (
	"net/http"
	"testing"
	"time"
)

func init() { jwtSecret = "test-secret" }

func TestIssueAndValidateToken(t *testing.T) {
	token := issueToken("admin", time.Hour)
	if !validToken(token) {
		t.Fatal("freshly issued token should be valid")
	}
}

func TestValidateTokenRejectsTamperedSignature(t *testing.T) {
	token := issueToken("admin", time.Hour)
	if validToken(token + "tampered") {
		t.Fatal("token with tampered signature must be rejected")
	}
}

func TestValidateTokenRejectsMalformed(t *testing.T) {
	cases := []string{"", "abc", "a.b", "a.b.c.d", "not.a.jwt"}
	for _, c := range cases {
		if validToken(c) {
			t.Fatalf("malformed token %q must be rejected", c)
		}
	}
}

func TestValidateTokenRejectsExpired(t *testing.T) {
	expired := issueToken("admin", -time.Hour)
	if validToken(expired) {
		t.Fatal("expired token must be rejected")
	}
}

func TestRequireAuth(t *testing.T) {
	token := issueToken("admin", time.Hour)

	withBearer, _ := http.NewRequest("PUT", "/api/v1/configs", nil)
	withBearer.Header.Set("Authorization", "Bearer "+token)
	if !requireAuth(withBearer) {
		t.Fatal("valid Bearer token should authorize")
	}

	noHeader, _ := http.NewRequest("PUT", "/api/v1/configs", nil)
	if requireAuth(noHeader) {
		t.Fatal("missing Authorization header must not authorize")
	}

	wrongScheme, _ := http.NewRequest("PUT", "/api/v1/configs", nil)
	wrongScheme.Header.Set("Authorization", "Basic "+token)
	if requireAuth(wrongScheme) {
		t.Fatal("non-Bearer scheme must not authorize")
	}
}
