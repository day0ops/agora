#!/bin/sh
# Queries the external brand registry for one listing. The X-Api-Key value
# below is a fixed placeholder, never the real key - the egress gateway's
# EgressPolicy.authentication swaps it for the real credential, which this
# script (and the model) never sees.
set -eu

listing_id=${1:?usage: check.sh <listing_id>}
registry_url="http://brand-registry.brand-registry.svc.cluster.local:8090"
errfile=$(mktemp)
trap 'rm -f "$errfile"' EXIT

body=$(wget -T 5 -qO- \
  --header 'X-Api-Key: kagent-credential-injected' \
  "$registry_url/v1/check?listing_id=$listing_id" 2>"$errfile") || {
  # Escape backslash and double-quote so wget's stderr can't produce invalid JSON.
  reason=$(tail -1 "$errfile" 2>/dev/null | sed 's/\\/\\\\/g; s/"/\\"/g')
  printf '{"status":"unavailable","reason":"%s"}\n' "${reason:-registry unreachable}"
  exit 0
}

echo "$body"
