package main

import (
	"slices"
	"strings"
	"testing"
)

// TestFilteredChildEnvStripsKubectlAteOwnReservedVars reproduces the finding
// that kubectl-ate itself reads ATE_NAMESPACE/ATE_API_SERVICE_NAME/
// ATE_CLIENT_SERVICE_ACCOUNT from its own process environment
// (solo-io/substrate's internal/ateclient/builder.go) - a name collision
// with any same-named variable this console sets for its own, unrelated
// purposes would silently redirect kubectl-ate's control-plane target.
func TestFilteredChildEnvStripsKubectlAteOwnReservedVars(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"ATE_NAMESPACE=kagent-system",
		"ATE_API_SERVICE_NAME=custom",
		"ATE_CLIENT_SERVICE_ACCOUNT=custom-sa",
		"HOME=/root",
	}
	got := filteredChildEnv(parent)
	for _, blocked := range []string{"ATE_NAMESPACE=", "ATE_API_SERVICE_NAME=", "ATE_CLIENT_SERVICE_ACCOUNT="} {
		for _, kv := range got {
			if strings.HasPrefix(kv, blocked) {
				t.Errorf("filteredChildEnv must strip %s, found %q", blocked, kv)
			}
		}
	}
	if !slices.Contains(got, "PATH=/usr/bin") || !slices.Contains(got, "HOME=/root") {
		t.Errorf("filteredChildEnv must preserve unrelated vars, got %v", got)
	}
}

func TestParseActorsJSON(t *testing.T) {
	data := []byte(`{"actors":[
		{"metadata":{"name":"a1-uuid","updateTime":"2026-11-22T10:00:00Z"},
		 "actorTemplate":{"name":"trust-safety-agora-swarm-1"},
		 "status":{"state":"ACTOR_STATE_RUNNING",
		   "workerAssignment":{"workerPool":"agora-workers","workerPod":"agora-workers-0"},
		   "snapshots":[{"generation":1}]}},
		{"metadata":{"name":"a2-uuid","updateTime":"2026-11-22T09:00:00Z"},
		 "actorTemplate":{"name":"auto-bidder-agora-swarm-1"},
		 "status":{"state":"ACTOR_STATE_SUSPENDED"}}
	]}`)
	actors, err := parseActorsJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actors) != 2 {
		t.Fatalf("expected 2 actors, got %d", len(actors))
	}
	if actors[0].Template != "trust-safety-agora-swarm-1" || actors[0].State != "running" {
		t.Errorf("actor 0 = %+v", actors[0])
	}
	if actors[0].LastActivity != "2026-11-22T10:00:00Z" {
		t.Errorf("actor 0 LastActivity = %q", actors[0].LastActivity)
	}
	if actors[0].Worker == nil || actors[0].Worker.WorkerPool != "agora-workers" || actors[0].Worker.WorkerPod != "agora-workers-0" {
		t.Errorf("actor 0 worker assignment = %+v", actors[0].Worker)
	}
	if !actors[0].HasSnapshot {
		t.Errorf("actor 0 should have a snapshot")
	}
	if actors[1].State != "suspended" || actors[1].Worker != nil {
		t.Errorf("actor 1 (suspended, no worker) = %+v", actors[1])
	}
	if actors[1].HasSnapshot {
		t.Errorf("actor 1 should have no snapshot")
	}
}

func TestParseActorsJSONInvalidReturnsError(t *testing.T) {
	_, err := parseActorsJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

// TestParseActorsJSONRejectsRecordWithMissingRequiredFields guards against
// silent kubectl-ate field-name drift (final review Review Focus #1): valid
// JSON with an unexpectedly different field name unmarshals without error,
// but leaves metadata.name/status.state as zero values - that must surface
// as a real error, not a blank/wrong-looking row in the UI.
func TestParseActorsJSONRejectsRecordWithMissingRequiredFields(t *testing.T) {
	data := []byte(`{"actors":[{"metadata":{},"actorTemplate":{"name":"trust-safety-1"},"status":{}}]}`)
	_, err := parseActorsJSON(data)
	if err == nil {
		t.Fatal("expected an error for a record with no metadata.name/status.state, got nil")
	}
}

func TestParseWorkerPodsJSON(t *testing.T) {
	data := []byte(`{"workers":[
		{"metadata":{"name":"w1"},"workerPool":"agora-workers","workerPod":"agora-workers-0",
		 "status":{"state":"WORKER_STATE_ACTIVE","allocated":{"actors":3},"capacity":{"actors":10}}},
		{"metadata":{"name":"w2"},"workerPool":"kagent-default","workerPod":"kagent-default-0",
		 "status":{"state":"WORKER_STATE_ACTIVE","allocated":{"actors":0},"capacity":{"actors":5}}}
	]}`)
	pods, err := parseWorkerPodsJSON(data, "agora-workers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod filtered to agora-workers, got %d", len(pods))
	}
	if pods[0].Name != "agora-workers-0" || pods[0].Pool != "agora-workers" {
		t.Errorf("pod 0 = %+v", pods[0])
	}
	if pods[0].State != "active" {
		t.Errorf("pod 0 state = %q, want active", pods[0].State)
	}
	if pods[0].Occupancy != "3/10" {
		t.Errorf("expected occupancy 3/10, got %q", pods[0].Occupancy)
	}
}

func TestParseWorkerPodsJSONInvalidReturnsError(t *testing.T) {
	_, err := parseWorkerPodsJSON([]byte(`not json`), "agora-workers")
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}
