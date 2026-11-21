package main

import "net/http"

type actorDTO struct {
	ID           string `json:"id"`       // the actor's own unique UUID - a React list key, never a display label
	Template     string `json:"template"` // the human-readable label to display (many actors can share one)
	State        string `json:"state"`
	Pool         string `json:"pool"`
	LastActivity string `json:"lastActivity"`
}

type workerDTO struct {
	Name      string `json:"name"`
	Pool      string `json:"pool"`
	State     string `json:"state"`
	Occupancy string `json:"occupancy"`
}

type substrateStatusResponse struct {
	Actors  []actorDTO  `json:"actors"`
	Workers []workerDTO `json:"workers"`
}

// handleSubstrateStatus serves Operator Studio's "Agent Substrate" panel:
// real actor/worker-pool status from kubectl-ate when configured, a 501
// ("not configured") when this deployment has no live substrate wired up
// (the adapter maps that to the existing honest-empty UI state), and a real
// 502 on an actual source failure - never an empty table indistinguishable
// from "zero actors are running".
func (s *consoleServer) handleSubstrateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	if id.Role != "operators" {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "operator role required"})
		return
	}
	if s.actorSource == nil {
		writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "substrate status not configured on this deployment"})
		return
	}
	actors, err := s.actorSource.ListActors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "listing actors: " + err.Error()})
		return
	}
	workers, err := s.actorSource.ListWorkerPods(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "listing worker pods: " + err.Error()})
		return
	}

	actorDTOs := make([]actorDTO, 0, len(actors))
	for _, a := range actors {
		pool := "unassigned"
		if a.Worker != nil {
			pool = a.Worker.WorkerPool
		}
		actorDTOs = append(actorDTOs, actorDTO{
			ID:           a.Name,
			Template:     a.Template,
			State:        a.State,
			Pool:         pool,
			LastActivity: a.LastActivity,
		})
	}
	workerDTOs := make([]workerDTO, 0, len(workers))
	for _, w := range workers {
		workerDTOs = append(workerDTOs, workerDTO(w))
	}

	writeJSON(w, http.StatusOK, substrateStatusResponse{Actors: actorDTOs, Workers: workerDTOs})
}
