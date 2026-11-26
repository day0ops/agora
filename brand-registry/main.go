package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

// seedVerdicts is this mock's entire "database" - a handful of catalog
// listings it actually has a record for, matching fixtures/seed.go's IDs.
// Every other listing ID comes back "unknown", on purpose: a real brand
// registry doesn't know about most of the world's second-hand goods either.
var seedVerdicts = map[string]Verdict{
	"listing-9": {
		Status: VerdictCounterfeit,
		Reason: "serial number range does not match Omega's registry for this model",
	},
	"listing-8": {
		Status: VerdictAuthentic,
		Reason: "serial number and body stamp match Fender's 1978 production records",
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("brand-registry: API_KEY must be set")
	}

	store := NewStore(seedVerdicts)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           NewMux(store, apiKey),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("brand-registry listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("brand-registry: %v", err)
	}
}
