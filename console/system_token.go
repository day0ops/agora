package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SystemTokenSource yields a bearer token for the console's own system
// identity - used for operator-triggered platform checks (the isolation
// probe), never for anything a visitor's own forwarded token should cover.
type SystemTokenSource interface {
	Token(ctx context.Context) (string, error)
}

// keycloakSystemTokenSource fetches a client-credentials access token and
// caches it until shortly before it expires. tokenURL must be Keycloak's
// PUBLIC hostname, never an internal Service DNS name - kagent-controller
// validates a token's `iss` claim against its own configured public issuer,
// the same rule documented for every other OIDC client in this repo.
type keycloakSystemTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	client       *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newKeycloakSystemTokenSource(tokenURL, clientID, clientSecret string) *keycloakSystemTokenSource {
	return &keycloakSystemTokenSource{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// expiryLeeway is how long before a cached token's real expiry it is
// refetched, so an in-flight request never races the token actually
// expiring mid-call.
const expiryLeeway = 30 * time.Second

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *keycloakSystemTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	if s.token != "" && time.Now().Before(s.expiresAt) {
		tok := s.token
		s.mu.Unlock()
		return tok, nil
	}
	s.mu.Unlock()

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch system token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read system token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch system token: HTTP %d: %s", resp.StatusCode, body)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode system token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("system token response had no access_token")
	}

	s.mu.Lock()
	s.token = tr.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - expiryLeeway)
	s.mu.Unlock()

	return tr.AccessToken, nil
}
