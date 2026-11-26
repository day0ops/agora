package main

import "sync/atomic"

// VerdictStatus is a brand-registry lookup's result for one listing.
// "unknown" means the registry has no record for that listing at all - this
// mock only ever answers for the handful of listings it was seeded with, the
// same honesty trust-mcp's reputation.go already follows: never fabricate a
// verdict this dataset has no way to actually support.
type VerdictStatus string

const (
	VerdictAuthentic   VerdictStatus = "authentic"
	VerdictCounterfeit VerdictStatus = "counterfeit_flagged"
	VerdictUnknown     VerdictStatus = "unknown"
)

// Verdict is one listing's registry record.
type Verdict struct {
	Status VerdictStatus `json:"status"`
	Reason string        `json:"reason,omitempty"`
}

// Stats is a point-in-time snapshot of request counters, independent of
// anything a caller claims - the demo's proof that a request did or did not
// actually reach this service.
type Stats struct {
	TotalRequests        int64 `json:"total_requests"`
	AuthorizedRequests   int64 `json:"authorized_requests"`
	UnauthorizedRequests int64 `json:"unauthorized_requests"`
}

// Store holds the registry's seeded verdicts and live request counters.
type Store struct {
	verdicts map[string]Verdict

	totalRequests        atomic.Int64
	authorizedRequests   atomic.Int64
	unauthorizedRequests atomic.Int64
}

func NewStore(verdicts map[string]Verdict) *Store {
	return &Store{verdicts: verdicts}
}

// Check looks up a listing's verdict, recording the request as authorized.
// Callers must gate this behind their own API key check first - Check itself
// does not re-verify authorization, only counts it.
func (s *Store) Check(listingID string) Verdict {
	s.totalRequests.Add(1)
	s.authorizedRequests.Add(1)
	if v, ok := s.verdicts[listingID]; ok {
		return v
	}
	return Verdict{Status: VerdictUnknown}
}

// RecordUnauthorized counts a request that was rejected for a missing or
// wrong API key, before any verdict lookup happened.
func (s *Store) RecordUnauthorized() {
	s.totalRequests.Add(1)
	s.unauthorizedRequests.Add(1)
}

func (s *Store) Stats() Stats {
	return Stats{
		TotalRequests:        s.totalRequests.Load(),
		AuthorizedRequests:   s.authorizedRequests.Load(),
		UnauthorizedRequests: s.unauthorizedRequests.Load(),
	}
}
