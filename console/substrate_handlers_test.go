package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeActorSource struct {
	actors  []Actor
	workers []WorkerPod
	err     error
}

func (f fakeActorSource) ListActors(context.Context) ([]Actor, error) { return f.actors, f.err }
func (f fakeActorSource) ListWorkerPods(context.Context) ([]WorkerPod, error) {
	return f.workers, f.err
}

func TestHandleSubstrateStatusRejectsNoIdentity(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.actorSource = fakeActorSource{actors: []Actor{{Name: "a1"}}}
	req := httptest.NewRequest("GET", "/account/substrate/status", nil)
	w := httptest.NewRecorder()
	s.handleSubstrateStatus(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleSubstrateStatusRejectsNonOperatorRole(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.actorSource = fakeActorSource{actors: []Actor{{Name: "a1"}}}
	req := httptest.NewRequest("GET", "/account/substrate/status", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	w := httptest.NewRecorder()
	s.handleSubstrateStatus(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-operator role, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleSubstrateStatusReturnsActorsAndWorkers(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.actorSource = fakeActorSource{
		actors: []Actor{
			{Name: "uuid-1", Template: "trust-safety-1", State: "running", LastActivity: "2026-11-22T10:00:00Z",
				Worker: &WorkerAssignment{WorkerPool: "agora-workers", WorkerPod: "agora-workers-0"}},
			{Name: "uuid-2", Template: "auto-bidder-1", State: "suspended"},
		},
		workers: []WorkerPod{{Name: "agora-workers-0", Pool: "agora-workers", State: "active", Occupancy: "3/10"}},
	}
	req := httptest.NewRequest("GET", "/account/substrate/status", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleSubstrateStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp substrateStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Actors) != 2 || resp.Actors[0].ID != "trust-safety-1" {
		t.Fatalf("response actors = %+v", resp.Actors)
	}
	if resp.Actors[0].Pool != "agora-workers" {
		t.Errorf("actor 0 pool = %q, want agora-workers", resp.Actors[0].Pool)
	}
	if resp.Actors[1].Pool != "unassigned" {
		t.Errorf("actor 1 (no worker) pool = %q, want unassigned", resp.Actors[1].Pool)
	}
	if len(resp.Workers) != 1 || resp.Workers[0].Occupancy != "3/10" {
		t.Fatalf("response workers = %+v", resp.Workers)
	}
}

func TestHandleSubstrateStatusNotConfiguredReturns501(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	// s.actorSource left nil -> "not configured" (local dev, honest-empty path)
	req := httptest.NewRequest("GET", "/account/substrate/status", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleSubstrateStatus(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestHandleSubstrateStatusRelaysSourceError(t *testing.T) {
	s := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	s.actorSource = fakeActorSource{err: context.DeadlineExceeded}
	req := httptest.NewRequest("GET", "/account/substrate/status", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	w := httptest.NewRecorder()
	s.handleSubstrateStatus(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 on a real source error, got %d: %s", w.Code, w.Body.String())
	}
}
