package main

import "testing"

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
