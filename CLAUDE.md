# CLAUDE.md

Guidance for Claude Code in this repository.

## Overview

Application tier for the marketplace demo on kagent Agent Substrate + agentgateway Enterprise: a live buy/sell + auction marketplace where buyer and seller AI concierges run as Substrate actors. Go monorepo (`go.work`): a `catalog-api` backend (owns all listings/bids/orders domain data, exposed as a REST API that doubles as the OpenAPI source agentgateway wraps into `listings-mcp` with no MCP server code), five purpose-built MCP servers, a `brand-registry` mock external service queried only by the `brand-check` Agent Skill (not an MCP server), a marketplace console (Go backend + web SPA with buyer/seller/operator views), and a shared synthetic `fixtures` module. Deployment glue lives in the separate `agentic-field-kit` repo.

## Conventions

- Go 1.27.0. MCP servers use `github.com/modelcontextprotocol/go-sdk` served at `POST /mcp`.
- Each component is its own module under `github.com/day0ops/agora-demo/`; add it to `go.work`.
- Shared data lives in `fixtures/` and is imported by servers, `catalog-api`, and the console; keep it the single source of truth (DRY).
- All data is synthetic. No real personal or financial data.
- `catalog-api`'s REST surface is the single source of truth for listings/bids/orders -- the purpose-built MCP servers (`bids-mcp`, `pricing-mcp`, `trust-mcp`, `orders-mcp`, `catalog-mcp`) call into it rather than holding their own copy of the data.

## Commands

The workspace has no root module, so a bare `./...` from the repo root matches nothing; `go list -m` expands to every module in `go.work`.

```bash
go work sync
go build $(go list -m -f '{{.Dir}}/...')
go test $(go list -m -f '{{.Dir}}/...')
golangci-lint run $(go list -m -f '{{.Dir}}/...')
```

## Style

- No em dash characters in prose; use a plain hyphen or restructure.
