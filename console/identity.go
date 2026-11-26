package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// identity is the console's view of the logged-in visitor, resolved from
// their forwarded Keycloak access token's claims.
type identity struct {
	Username   string   `json:"username"`
	Groups     []string `json:"groups"`
	Role       string   `json:"role"`
	FixtureID  string   `json:"fixtureId,omitempty"`
	SellerTier string   `json:"sellerTier,omitempty"`
}

// rolePriority orders the three marketplace groups by precedence, highest
// first, for a visitor who ends up in more than one. None of this phase's
// demo users are multi-group; a role-switcher UI has nothing to exercise
// yet, so the active role is just whichever of these ranks highest.
var rolePriority = []string{"operators", "sellers", "buyers"}

func highestPriorityRole(groups []string) string {
	for _, want := range rolePriority {
		for _, g := range groups {
			if g == want {
				return want
			}
		}
	}
	return ""
}

// forwardedClaims is the subset of a Keycloak access token's JWT claims the
// console reads. oidc-gateway's ext-auth-service already validated the
// token's signature and expiry before forwarding it -- the console only
// ever parses the payload, never re-verifies it.
//
// FixtureID and SellerTier come from Keycloak custom-attribute claims (see
// the 'kagent' realm's customAttributes in agentic-field-kit's profile
// YAML) rather than any mapping hardcoded here -- the same fixture_id claim
// agentgateway's agora-mcp-bid-identity CEL policy already checks against
// place_bid/set_autobid_max calls, so the console and the gateway agree on
// one source of truth instead of keeping their own copies in sync.
type forwardedClaims struct {
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"Groups"`
	FixtureID         string   `json:"fixture_id"`
	SellerTier        string   `json:"seller_tier"`
}

func decodeForwardedClaims(token string) (*forwardedClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token: expected 3 segments, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}
	var claims forwardedClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal token claims: %w", err)
	}
	return &claims, nil
}

// resolveIdentity reads the visitor's identity from their forwarded access
// token. Returns ErrNotAuthenticated if the header is absent -- oidc-gateway
// only reaches this far once a real session exists, so an absent header here
// means this specific request bypassed the gate, not that login is needed.
func (s *consoleServer) resolveIdentity(r *http.Request) (*identity, error) {
	token := r.Header.Get(forwardedTokenHeader)
	if token == "" {
		return nil, ErrNotAuthenticated
	}
	claims, err := decodeForwardedClaims(token)
	if err != nil {
		return nil, fmt.Errorf("resolve identity: %w", err)
	}
	return &identity{
		Username:   claims.PreferredUsername,
		Groups:     claims.Groups,
		Role:       highestPriorityRole(claims.Groups),
		FixtureID:  claims.FixtureID,
		SellerTier: claims.SellerTier,
	}, nil
}
