package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAPIKey = "test-key"

func newTestServer(t *testing.T) (*httptest.Server, *Store) {
	t.Helper()
	store := NewStore(map[string]Verdict{
		"listing-9": {Status: VerdictCounterfeit, Reason: "serial mismatch"},
	})
	srv := httptest.NewServer(NewMux(store, testAPIKey))
	t.Cleanup(srv.Close)
	return srv, store
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return v
}

func TestHandleCheckRejectsMissingAPIKey(t *testing.T) {
	srv, store := newTestServer(t)

	resp, err := http.Get(srv.URL + "/v1/check?listing_id=listing-9")
	if err != nil {
		t.Fatalf("GET /v1/check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if stats := store.Stats(); stats.UnauthorizedRequests != 1 || stats.AuthorizedRequests != 0 {
		t.Fatalf("unexpected stats after missing key: %+v", stats)
	}
}

func TestHandleCheckRejectsWrongAPIKey(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/check?listing_id=listing-9", nil)
	req.Header.Set("X-Api-Key", "wrong-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleCheckReturnsSeededVerdict(t *testing.T) {
	srv, store := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/check?listing_id=listing-9", nil)
	req.Header.Set("X-Api-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	type checkResponse struct {
		ListingID string `json:"listing_id"`
		Status    string `json:"status"`
		Reason    string `json:"reason"`
	}
	got := decodeJSON[checkResponse](t, resp)
	if got.Status != string(VerdictCounterfeit) {
		t.Fatalf("expected counterfeit_flagged, got %q", got.Status)
	}
	if stats := store.Stats(); stats.AuthorizedRequests != 1 {
		t.Fatalf("expected one authorized request, got %+v", stats)
	}
}

func TestHandleCheckReturnsUnknownForAnUnseededListing(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/check?listing_id=listing-404", nil)
	req.Header.Set("X-Api-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	type checkResponse struct {
		Status string `json:"status"`
	}
	got := decodeJSON[checkResponse](t, resp)
	if got.Status != string(VerdictUnknown) {
		t.Fatalf("expected unknown, got %q", got.Status)
	}
}

func TestHandleCheckRejectsMissingListingID(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/check", nil)
	req.Header.Set("X-Api-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandleStatsIsUnauthenticated(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/stats")
	if err != nil {
		t.Fatalf("GET /stats: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	got := decodeJSON[Stats](t, resp)
	if got.TotalRequests != 0 {
		t.Fatalf("expected a fresh store to report zero requests, got %+v", got)
	}
}
