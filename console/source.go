package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// WorkerAssignment is the worker pod/pool an Actor is currently placed on.
// Unset (nil on Actor.Worker) whenever the Actor has no worker (e.g.
// suspended - "parked").
type WorkerAssignment struct {
	WorkerPod  string
	WorkerPool string
}

// Actor is the console's own view of one kubectl-ate actor record, reduced
// to the fields Operator Studio renders.
type Actor struct {
	Name         string // metadata.name - a UUID, not human-readable; kept as an internal key only
	Template     string // actorTemplate.name - the human-readable label to display
	State        string // normalized status.state, e.g. "running", "suspended" (parked)
	LastActivity string // metadata.updateTime - a real server timestamp, last state-change time
	HasSnapshot  bool   // len(status.snapshots) > 0
	Worker       *WorkerAssignment
}

// WorkerPod is the console's own view of one kubectl-ate worker record.
type WorkerPod struct {
	Name      string
	Pool      string
	State     string
	Occupancy string // e.g. "3/10", from status.allocated.actors / status.capacity.actors
}

// ActorSource yields the live swarm actors and worker pods for Operator
// Studio's Agent Substrate panel.
type ActorSource interface {
	ListActors(ctx context.Context) ([]Actor, error)
	ListWorkerPods(ctx context.Context) ([]WorkerPod, error)
}

// kubectlAteActorSource shells out to the kubectl-ate CLI (bundled in the
// console image, see the Dockerfile) rather than hand-writing an ate-api
// gRPC client - the same technique the sibling fraud-ops-console demo
// proved live. kubectl-ate authenticates as the `ate-client` ServiceAccount
// via its own TokenRequest; this feature's own ServiceAccount must be
// granted `serviceaccounts/token: create` on `ate-client` in ate-system
// (see the agora-console feature's RBAC).
//
// Field shapes below are grounded in solo-io/substrate's
// pkg/proto/ateapipb/ateapi.proto (read directly from the checked-out
// source during design), not yet confirmed against a live, successfully
// authenticated call - run `kubectl-ate get actors --atespace <namespace>
// -o json` for real once this is deployed, and adjust the struct tags here
// if the live response differs in any field name or nesting.
type kubectlAteActorSource struct {
	binPath  string // path to the bundled kubectl-ate binary
	endpoint string // e.g. api.ate-system.svc.cluster.local:443
	atespace string // the kagent namespace agora-workers/the agents' actors live in -
	// deliberately not named "namespace" to avoid any reader conflating this with
	// ATE_NAMESPACE, which is a different, reserved concept kubectl-ate itself
	// reads from its own process environment (see kubectlAteReservedEnvVars)
	workerPoolName string // e.g. agora-workers -- filters ListWorkerPods to this pool only
}

func newKubectlAteActorSource(binPath, endpoint, atespace, workerPoolName string) *kubectlAteActorSource {
	return &kubectlAteActorSource{binPath: binPath, endpoint: endpoint, atespace: atespace, workerPoolName: workerPoolName}
}

type ateMetadata struct {
	Name       string `json:"name"`
	UpdateTime string `json:"updateTime"`
}

type ateObjectRef struct {
	Name string `json:"name"`
}

type ateWorkerAssignment struct {
	WorkerPod  string `json:"workerPod"`
	WorkerPool string `json:"workerPool"`
}

type ateSnapshot struct {
	Generation int `json:"generation"`
}

type ateActorStatus struct {
	State            string               `json:"state"`
	WorkerAssignment *ateWorkerAssignment `json:"workerAssignment"`
	Snapshots        []ateSnapshot        `json:"snapshots"`
}

// ateActorRecord is kubectl-ate's protojson-marshaled Actor message
// (ateapipb.Actor) - metadata.name is a UUID, actorTemplate.name is the
// human-readable template/harness reference to display instead.
type ateActorRecord struct {
	Metadata      ateMetadata    `json:"metadata"`
	ActorTemplate ateObjectRef   `json:"actorTemplate"`
	Status        ateActorStatus `json:"status"`
}

type ateActorsResponse struct {
	Actors []ateActorRecord `json:"actors"`
}

type ateResourceCount struct {
	Actors int `json:"actors"`
}

type ateWorkerStatus struct {
	State     string           `json:"state"`
	Allocated ateResourceCount `json:"allocated"`
	Capacity  ateResourceCount `json:"capacity"`
}

// ateWorkerRecord is kubectl-ate's protojson-marshaled Worker message
// (ateapipb.Worker) - workerPool/workerPod are top-level fields, unlike the
// nested WorkerAssignment on an Actor's own status.
type ateWorkerRecord struct {
	Metadata   ateMetadata     `json:"metadata"`
	WorkerPool string          `json:"workerPool"`
	WorkerPod  string          `json:"workerPod"`
	Status     ateWorkerStatus `json:"status"`
}

type ateWorkersResponse struct {
	Workers []ateWorkerRecord `json:"workers"`
}

// kubectlAteReservedEnvVars are read by kubectl-ate itself, straight out of
// its own process environment, to locate the substrate control plane
// (solo-io/substrate's internal/ateclient/builder.go: NamespaceEnv,
// APIServiceEnv, ClientServiceAccountEnv). Any same-named variable this
// console process happens to carry for its own purposes would silently
// override kubectl-ate's target instead of just being ignored - confirmed
// live during review: ATE_NAMESPACE set to this console's own namespace
// redirected kubectl-ate's TokenRequest/TLS-SNI/port-forward target away
// from the real ate-system control plane. filteredChildEnv strips exactly
// these three keys from the inherited environment before every exec, so
// kubectl-ate always falls back to its own correct built-in defaults
// regardless of what this process's environment contains now or later.
var kubectlAteReservedEnvVars = []string{
	"ATE_NAMESPACE=",
	"ATE_API_SERVICE_NAME=",
	"ATE_CLIENT_SERVICE_ACCOUNT=",
}

func filteredChildEnv(parent []string) []string {
	filtered := make([]string, 0, len(parent))
	for _, kv := range parent {
		reserved := false
		for _, prefix := range kubectlAteReservedEnvVars {
			if strings.HasPrefix(kv, prefix) {
				reserved = true
				break
			}
		}
		if !reserved {
			filtered = append(filtered, kv)
		}
	}
	return filtered
}

func (s *kubectlAteActorSource) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, s.binPath, append(args, "--endpoint", s.endpoint, "-o", "json")...) //nolint:gosec // fixed binary, fixed args
	cmd.Env = filteredChildEnv(os.Environ())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl-ate %v: %w: %s", args, err, stderr.String())
	}
	return out, nil
}

// ateStateToState normalizes ate's ACTOR_STATE_RUNNING-style enum values to
// the lowercase form the console frontend expects (e.g. "suspended" is the
// "parked" state this feature exists to show).
func ateStateToState(s string) string {
	s = strings.TrimPrefix(s, "ACTOR_STATE_")
	s = strings.TrimPrefix(s, "WORKER_STATE_")
	return strings.ToLower(s)
}

// ListActors scopes to this feature's own atespace (namespace), matching
// --atespace <namespace> - deliberately excluding the golden-snapshot
// atespace, whose actors are always suspended and are not real sessions.
func (s *kubectlAteActorSource) ListActors(ctx context.Context) ([]Actor, error) {
	out, err := s.run(ctx, "get", "actors", "--atespace", s.atespace)
	if err != nil {
		return nil, err
	}
	return parseActorsJSON(out)
}

// parseActorsJSON is the pure half of ListActors, split out so it's
// testable without shelling out to kubectl-ate.
func parseActorsJSON(data []byte) ([]Actor, error) {
	var resp ateActorsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing kubectl-ate actors output: %w", err)
	}
	actors := make([]Actor, 0, len(resp.Actors))
	for _, a := range resp.Actors {
		// metadata.name/status.state are required on every real actor
		// record - valid-but-unexpectedly-shaped JSON (e.g. a field renamed
		// in a newer kubectl-ate version) unmarshals without error but
		// leaves these as zero values, which must surface as a real parse
		// error, not a blank/wrong-looking row in the UI.
		if a.Metadata.Name == "" || a.Status.State == "" {
			return nil, fmt.Errorf("kubectl-ate actor record missing metadata.name or status.state (field-name drift?): %+v", a)
		}
		actor := Actor{
			Name:         a.Metadata.Name,
			Template:     a.ActorTemplate.Name,
			State:        ateStateToState(a.Status.State),
			LastActivity: a.Metadata.UpdateTime,
			HasSnapshot:  len(a.Status.Snapshots) > 0,
		}
		if a.Status.WorkerAssignment != nil {
			actor.Worker = &WorkerAssignment{
				WorkerPod:  a.Status.WorkerAssignment.WorkerPod,
				WorkerPool: a.Status.WorkerAssignment.WorkerPool,
			}
		}
		actors = append(actors, actor)
	}
	return actors, nil
}

// ListWorkerPods returns every worker pod in this feature's own
// workerPoolName - `get workers -n <namespace>` returns every pool sharing
// that namespace (e.g. kagent-default alongside agora-workers), so this
// filters by pool client-side.
func (s *kubectlAteActorSource) ListWorkerPods(ctx context.Context) ([]WorkerPod, error) {
	out, err := s.run(ctx, "get", "workers", "-n", s.atespace)
	if err != nil {
		return nil, err
	}
	return parseWorkerPodsJSON(out, s.workerPoolName)
}

// parseWorkerPodsJSON is the pure half of ListWorkerPods, split out so it's
// testable without shelling out to kubectl-ate.
func parseWorkerPodsJSON(data []byte, workerPoolName string) ([]WorkerPod, error) {
	var resp ateWorkersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing kubectl-ate workers output: %w", err)
	}
	pods := make([]WorkerPod, 0, len(resp.Workers))
	for _, w := range resp.Workers {
		if w.WorkerPool != workerPoolName {
			continue
		}
		pods = append(pods, WorkerPod{
			Name:      w.WorkerPod,
			Pool:      w.WorkerPool,
			State:     ateStateToState(w.Status.State),
			Occupancy: fmt.Sprintf("%d/%d", w.Status.Allocated.Actors, w.Status.Capacity.Actors),
		})
	}
	return pods, nil
}
