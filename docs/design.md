# Agora Design Notes

Architecture and API reference for the Agora application tier. Read the [README](../README.md) first; this doc covers the `catalog-api` and console API surfaces, the why behind trust scoring, how the console talks to live agents, the console's design system, and the agent-observability surfaces.

## catalog-api

```
GET    /listings                     list listings, filterable by ?status= and ?sellerId=
GET    /listings/{id}                get a listing
POST   /listings                     create a listing
GET    /listings/{id}/bids           bid history for a listing
POST   /listings/{id}/bids           place a manual bid
DELETE /listings/{id}/bids/{bidID}   withdraw a specific bid
POST   /listings/{id}/autobid        set (or raise) a buyer's standing proxy-bid ceiling
POST   /listings/{id}/checkout       buy a buy-now listing, or claim a won auction
GET    /orders/{id}                  get an order
GET    /orders                       list orders, filterable by ?buyerId= and ?sellerId=
```

Auctions close automatically once their end time passes: to `sold` if they received a bid, `unsold-expired` otherwise. Proxy bidding is classic eBay-style. The price settles one increment above the second-highest standing ceiling, capped at the top ceiling. A manual bid also counts as a ceiling of exactly its own amount, so a stronger standing auto-bid immediately outbids it. Checking out a won auction requires the caller to be the winning bidder (403 otherwise), and each listing can only be checked out once.

## console

The console is a backend-for-frontend over `catalog-api` plus a live SSE feed, with no marketplace data of its own. Public read endpoints live under `/api`; everything tied to a user lives under `/account`, which `oidc-gateway` gates behind a Keycloak session.

```
GET    /api/listings                         list listings, filterable by ?status= and ?sellerId=
GET    /api/listings/{id}                    get a listing
GET    /api/listings/{id}/bids               bid history for a listing
GET    /api/events                           live SSE feed of marketplace activity
GET    /api/version                          console build version

GET    /account/me                           resolved identity and role of the caller
GET    /account/login                        trigger the login redirect, then return to ?returnTo=
POST   /account/listings                     create a listing as the caller
POST   /account/listings/{id}/bids           place a manual bid as the caller
DELETE /account/listings/{id}/bids/{bidID}   withdraw a specific bid
POST   /account/listings/{id}/autobid        set (or raise) the caller's proxy-bid ceiling
POST   /account/listings/{id}/checkout       buy a buy-now listing, or claim a won auction
GET    /account/orders                       the caller's orders
POST   /account/chat                         send a message to a named concierge agent
GET    /account/stats                        active listings / total orders / connected clients (operators only)
```

The caller's buyer or seller ID always comes from the resolved identity, never from the request body. Write endpoints relay `catalog-api`'s own status code and error message rather than flattening every failure to 400, so the frontend can tell "not found" from "conflict" from "forbidden". Every successful write publishes an event onto the SSE feed (`listing.created`, `bid.placed`, `bid.withdrawn`, `autobid.set`, `order.created`), which all three views subscribe to for live updates.

The frontend is a single-page React app (`react-router-dom` for the landing, shop, sell, and ops routes), built to `console/web/dist` and embedded into the Go binary via `//go:embed`. Tests run on Vitest and React Testing Library. There is no demo identity picker: each view resolves the caller from `GET /account/me`, which reads the Keycloak access token `oidc-gateway` forwards. A logged-out visit triggers the gate's redirect to Keycloak, and the resolved role (buyer, seller, or operator) drives which view renders.

`POST /account/chat` reaches live kagent agents over A2A through agentgateway, authenticated as the logged-in user when `AGENTGATEWAY_BASE_URL` is set; see Agent invocation below for how that works and how it falls back when it's unset. See Design system, Execution drawer, and Operator Studio below for how the UI looks and behaves.

## Trust scoring and the Jev extension point

Bid risk scoring lives in its own module, `trust-scoring/`, so the same logic backs two callers:

- `catalog-api` calls it in-process on every bid, recording the result as the bid's trust check.
- `trust-mcp` exposes it to agents as the `score_bid_risk` tool.

The design calls for this scoring to eventually run on Jev, TypeSafe.ai's fast "System One" structured-decision model. There is no confirmed Jev API contract, credentials, or endpoint available to this project, only a blog post. Rather than ship an integration against an unverified API shape, `trust-scoring/scorer.go` defines a `RiskScorer` interface with a working, tested local heuristic as the default. The heuristic deliberately flags only explainable patterns computable from a single listing's own bid history.

> [!NOTE]
> `NewScorer()` in `trust-scoring/scorer.go` is the swap point for a Jev-backed scorer. `trust-mcp` uses it, but `catalog-api` currently constructs `NewHeuristicScorer()` directly (`catalog-api/store.go`), so swapping in Jev means changing that call site too.

## Agent invocation

The console's chat panels talk to real kagent concierge agents (for example `shopping-assistant` and `listing-pricing-assistant`). `console/agent_invoker.go` defines the seam:

```go
type AgentInvoker interface {
	Send(ctx context.Context, agentName, sessionID, message, token string) (reply string, err error)
}
```

Two implementations exist, selected at startup in `console/main.go`:

| Implementation | Selected when | Behavior |
|---|---|---|
| `AgentInvokerStub` | `AGENTGATEWAY_BASE_URL` unset (default) | Returns `ErrAgentInvokerNotConfigured` ("agent invocation is not wired to a live kagent cluster yet") for every call. The chat panel shows that message. |
| `AgentInvokerHTTP` | `AGENTGATEWAY_BASE_URL` set | Calls kagent-controller's A2A JSON-RPC `SendMessage` method through agentgateway at `<baseURL>/<agentName>`. Verified working against a live kagent-enterprise 2026.11.1 cluster. |

`AgentInvokerHTTP` authenticates as the logged-in user: it forwards the caller's own Keycloak access token (received from `oidc-gateway` in the `X-Agora-User-Token` header) as `Authorization: Bearer <token>`. There is no service-account identity. Because nothing in the A2A payload carries identity, `POST /account/chat` prefixes each message with the resolved `[authenticated_user_id: ..., role: ...]`, and each agent's system prompt must treat that as the only valid ID for its own tool calls.

`AGENTGATEWAY_BASE_URL` must already include kagent-controller's path prefix (`/agents/<namespace>`). Without it, the request falls through to whatever other route shares the Gateway, typically the console's own catch-all route, which answers with the SPA shell and a 200.

> [!IMPORTANT]
> Each of these A2A 1.0 details was confirmed live against kagent-enterprise 2026.11.1 and is easy to regress:
>
> - **Method name** is `SendMessage` (PascalCase), not the A2A spec's `message/send`.
> - **Protocol version** requires an `A2A-Version: 1.0` header and a `configuration.acceptedOutputModes` field. Without both, the server assumes protocol 0.3 and rejects the request with "version not supported".
> - **`contextId` is server-assigned.** Omit it on a session's first message so kagent creates a fresh context, then cache the returned ID per console session for later turns. A client-invented `contextId` fails with JSON-RPC error `-32001` "task not found".
> - **The reply** lives in the last artifact's text parts (`task.artifacts[-1].parts[].text`), not in a flat status message field.

## Design system

The console is branded **Agora**.

- **Tokens** (`console/web/src/style.css`, `TOKENS` and `BASE` sections): one `:root` block plus an `html[data-theme="light"]` override drive both themes from the same palette and spacing/radius scale. Both themes are verified against WCAG AA for text and status-pill contrast. Fonts are Geist Variable (sans) and JetBrains Mono Variable (prices, lot numbers, timers, IDs), loaded from the Fontsource CDN.
- **Icons** (`console/web/src/icons.tsx`): one inline SVG set (`currentColor`-themeable) for the brand mark, nav, theme toggle, and UI affordances. No emoji anywhere in the app.
- **Components** (`console/web/src/components/`): `Navbar` and `Footer`, `LotCard` (the imagery-led listing card shared by Shop and Sell), `ChatPanel` (the collapsible assistant panel), `ListingImage`, `StatusPill`, `EmptyState`, `NotConnectedBanner`, and `ExecutionDrawer` with its `ExecutionTimeline` and `TechnicalInspector` sections (see below). Countdown timers come from the `useCountdown` hook.
- **Imagery** (`console/web/public/images/`): each seeded listing has a real public-domain or CC0 photo from Wikimedia Commons, with license and author recorded per image in `public/images/ATTRIBUTIONS.md`. Any listing without a committed photo, including every listing created live through the Sell form, falls back to a fixed-color per-category SVG (`public/images/categories/`), never a broken image.
- **Accessibility**: semantic landmarks on every page, a visible `:focus-visible` outline, keyboard-operable lot cards, labeled form controls (visually hidden labels where a visible one would clutter a compact control), and `prefers-reduced-motion` respected throughout.

> [!IMPORTANT]
> **Honesty rule:** nothing in the console is fabricated. Where real data does not exist yet, the UI says so with a plain empty state instead of inventing it. The Sell page's "Recommended pricing" and "Comparable items" sections explain they populate once a live agent is connected, and the chat panels surface the stub's not-configured error verbatim when no cluster is wired in. New surfaces must follow the same rule.

## Execution drawer

Every sold listing, order, and relevant activity-feed entry on the Shop and Ops views has a "Behind the scenes" trigger (`console/web/src/components/ExecutionDrawer.tsx`) that opens a focus-trapped, three-level drawer:

- **Summary** is real today. It shows the trust check `catalog-api` ran in-process when the bid was placed: score and flagged reasons, or "not available" for bids that predate the trust check.
- **Agent Journey** and **Technical Inspector** are built and honestly empty. Both are wired to `console/web/src/adapters/executionAdapter.ts` and `substrateAdapter.ts`, which always return an empty result today because nothing populates a bid's or order's correlation ID yet. With no live Agent Substrate run to assemble a journey or inspector trace from, both render a plain "not available" state rather than fabricated steps, and need no caller changes once a real adapter is wired in.

Keyboard behavior is verified: `role="dialog"` with `aria-modal`, focus moves into the drawer on open and is trapped within it on Tab and Shift+Tab, and Escape closes the drawer and restores focus to the trigger that opened it.

## Operator Studio: Agent Substrate panel

The Ops view (`/ops`) has an "Agent Substrate" panel (`console/web/src/components/SubstrateTopology.tsx`) showing the column shape a live actor-status table will have: Actor, State, Pool, Last activity. Like the drawer's Technical Inspector, it is honestly empty today. `getSubstrateStatus()` in `substrateAdapter.ts` always resolves "empty" because no live Agent Substrate cluster is wired to the console yet, so the body renders a not-connected banner spanning all four columns instead of fabricated rows. It needs no caller changes once a real adapter is wired in, and is verified responsive at desktop (1280px) and mobile (390px) widths with no overflow or clipping.
