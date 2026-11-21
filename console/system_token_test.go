package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeycloakSystemTokenSourceFetchesAndCaches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "expires_in": 300})
	}))
	defer srv.Close()
	src := newKeycloakSystemTokenSource(srv.URL, "agora-console-system", "secret")
	tok1, err := src.Token(context.Background())
	if err != nil || tok1 != "tok-1" {
		t.Fatalf("token1 = %q, err = %v", tok1, err)
	}
	tok2, err := src.Token(context.Background())
	if err != nil || tok2 != "tok-1" || calls != 1 {
		t.Fatalf("expected cached token with 1 call, got tok=%q calls=%d err=%v", tok2, calls, err)
	}
}

func TestKeycloakSystemTokenSourcePropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	src := newKeycloakSystemTokenSource(srv.URL, "agora-console-system", "wrong-secret")
	if _, err := src.Token(context.Background()); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestKeycloakSystemTokenSourceRefetchesAfterExpiry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "expires_in": 0})
	}))
	defer srv.Close()
	src := newKeycloakSystemTokenSource(srv.URL, "agora-console-system", "secret")
	if _, err := src.Token(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := src.Token(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected a refetch once the cached token's expiry has passed, got %d calls", calls)
	}
}
