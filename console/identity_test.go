package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

func fakeToken(t *testing.T, username string, groups []string, fixtureID, sellerTier string) string {
	t.Helper()
	payload, err := json.Marshal(forwardedClaims{
		PreferredUsername: username,
		Groups:            groups,
		FixtureID:         fixtureID,
		SellerTier:        sellerTier,
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestResolveIdentityReturnsErrNotAuthenticatedWithNoHeader(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)

	if _, err := srv.resolveIdentity(req); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestResolveIdentityMapsDemoBuyerToBuyerOneFixture(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}, "buyer-1", ""))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "demo-buyer" || id.Role != "buyers" || id.FixtureID != "buyer-1" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestResolveIdentityMapsDemoBuyer2ToBuyerTwoFixture(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer2", []string{"buyers"}, "buyer-2", ""))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "demo-buyer2" || id.Role != "buyers" || id.FixtureID != "buyer-2" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestResolveIdentityPicksHighestPriorityRoleWhenMultipleGroupsPresent(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "someone", []string{"buyers", "operators"}, "", ""))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Role != "operators" {
		t.Fatalf("expected operators to outrank buyers, got role %q", id.Role)
	}
}

func TestResolveIdentityLeavesFixtureIDEmptyWhenTokenCarriesNoClaim(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}, "", ""))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.FixtureID != "" {
		t.Fatalf("expected no fixture_id for a token with no fixture_id claim, got %q", id.FixtureID)
	}
}

func TestResolveIdentityPassesThroughSellerTierClaim(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers", "sellers"}, "buyer-1", "pro"))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.SellerTier != "pro" {
		t.Fatalf("expected the token's seller_tier claim to pass through, got %q", id.SellerTier)
	}
}

func TestResolveIdentityLeavesSellerTierEmptyWhenTokenCarriesNoClaim(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}, "buyer-1", ""))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.SellerTier != "" {
		t.Fatalf("expected no sellerTier for a buyer, got %q", id.SellerTier)
	}
}

func TestResolveIdentityRejectsAMalformedToken(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, "not-a-jwt")

	if _, err := srv.resolveIdentity(req); err == nil {
		t.Fatal("expected an error for a malformed token")
	}
}
