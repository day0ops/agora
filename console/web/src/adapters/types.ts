// AdapterResult is the contract every Phase 2/3 data adapter returns.
// Components switch on `status` - never infer "real" from a merely non-null
// value - so an adapter can go from always-empty to real with no caller
// changes. See docs/superpowers/specs/2026-10-30-agora-ui-redesign-design.md
// section 17.2. "error" is a real, configured data source that failed on
// this call (network error, non-2xx) - distinct from "empty" (this
// deployment has no live source configured at all) - a component must never
// render one as the other, per the honesty rule.
export type AdapterResult<T> =
  | { status: "real"; data: T }
  | { status: "empty"; reason: string }
  | { status: "error"; message: string };
