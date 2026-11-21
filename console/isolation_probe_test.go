package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeProbeInvoker (distinct from handlers_test.go's own fakeInvoker, which
// is specific to the visitor chat path) also records every session ID it
// was called with, to assert freshness across repeated probe calls.
type fakeProbeInvoker struct {
	reply string
	err   error
	calls []string
}

func (f *fakeProbeInvoker) Send(_ context.Context, _, sessionID, _, _ string) (string, error) {
	f.calls = append(f.calls, sessionID)
	return f.reply, f.err
}

type fakeTokenSource struct {
	token string
	err   error
}

func (f fakeTokenSource) Token(context.Context) (string, error) { return f.token, f.err }

func TestHandleIsolationCheckRejectsNoIdentity(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	body := strings.NewReader(`{"target":"trust-safety"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleIsolationCheckRejectsNonOperatorRole(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	body := strings.NewReader(`{"target":"trust-safety"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-operator role, got %d", w.Code)
	}
}

func TestHandleIsolationCheckRejectsUnknownTarget(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	body := strings.NewReader(`{"target":"shopping-assistant"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a non-probeable target, got %d", w.Code)
	}
}

func TestHandleIsolationCheckNotConfiguredReturns501(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	body := strings.NewReader(`{"target":"trust-safety"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestHandleIsolationCheckUsesFreshSessionEachCall(t *testing.T) {
	inv := &fakeProbeInvoker{reply: "I don't have access to order history."}
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.probeInvoker = inv
	s.systemToken = fakeTokenSource{token: "sys-tok"}
	for i := 0; i < 2; i++ {
		body := strings.NewReader(`{"target":"trust-safety"}`)
		req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
		req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
		w := httptest.NewRecorder()
		s.handleIsolationCheck(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d: %s", i, w.Code, w.Body.String())
		}
	}
	if len(inv.calls) != 2 || inv.calls[0] == inv.calls[1] {
		t.Fatalf("expected 2 distinct session IDs, got %v", inv.calls)
	}
}

func TestHandleIsolationCheckReturnsReply(t *testing.T) {
	inv := &fakeProbeInvoker{reply: "I can't help with pricing - I don't have that tool."}
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.probeInvoker = inv
	s.systemToken = fakeTokenSource{token: "sys-tok"}
	body := strings.NewReader(`{"target":"auto-bidder"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp isolationCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Reply != inv.reply {
		t.Errorf("reply = %q, want %q", resp.Reply, inv.reply)
	}
}

func TestHandleIsolationCheckMapsAuthFailureHonestly(t *testing.T) {
	inv := &fakeProbeInvoker{err: ErrNotAuthenticated}
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.probeInvoker = inv
	s.systemToken = fakeTokenSource{token: "sys-tok"}
	body := strings.NewReader(`{"target":"auto-bidder"}`)
	req := httptest.NewRequest("POST", "/account/substrate/isolation-check", body)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleIsolationCheck(w, req)
	var resp errorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if strings.Contains(strings.ToLower(resp.Error), "log in") {
		t.Errorf("probe auth failure must not read like a login prompt, got %q", resp.Error)
	}
}
