package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// NewMux wires the brand registry's HTTP routes. apiKey is the exact value
// GET /v1/check requires in its X-Api-Key header - GET /stats is
// deliberately unauthenticated, since its whole purpose is to let a caller
// outside the gated path (e.g. Agora's own operator console) prove whether a
// request got through, independent of the key itself.
func NewMux(store *Store, apiKey string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/check", handleCheck(store, apiKey))
	mux.HandleFunc("GET /stats", handleStats(store))
	return mux
}

func handleCheck(store *Store, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != apiKey {
			store.RecordUnauthorized()
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or missing API key"})
			return
		}
		listingID := r.URL.Query().Get("listing_id")
		if listingID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "listing_id is required"})
			return
		}
		verdict := store.Check(listingID)
		writeJSON(w, http.StatusOK, struct {
			ListingID string `json:"listing_id"`
			Verdict
		}{ListingID: listingID, Verdict: verdict})
	}
}

func handleStats(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.Stats())
	}
}
