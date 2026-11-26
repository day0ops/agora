---
name: brand-check
description: Verify a listing's brand/authenticity against the external brand registry. Use when screening a listing or bid for counterfeit signals and you need a real registry verdict, not a guess.
---

Run `/skills/brand-check/check.sh <listing_id>` (the full absolute path, not
a relative one) via your shell tool to query the external brand registry for
one listing ID. It prints one line of JSON to stdout:

- `{"status":"authentic","reason":"..."}` - the registry has a positive record.
- `{"status":"counterfeit_flagged","reason":"..."}` - the registry flagged this listing.
- `{"status":"unknown"}` - the registry has no record for this listing.
- `{"status":"unavailable","reason":"..."}` - the registry could not be reached
  (for example, network access to it is not currently granted).

Never state a brand/authenticity verdict you did not get from this script's
output. If the result is "unavailable", say verification is unavailable right
now - do not guess, and do not retry more than once.
