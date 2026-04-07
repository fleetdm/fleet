# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About Fleet

Fleet is an open-source platform for IT and security teams: device management (MDM), vulnerability reporting, osquery fleet management, and security monitoring. Go backend, React/TypeScript frontend, manages thousands of devices across macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS.

## Architecture

### Backend request flow
HTTP request → `server/service/handler.go` routes → endpoint function (decode request) → service method (auth + business logic) → datastore method (SQL) → response struct

### Key layers
- **Types & interfaces**: `server/fleet/` — `Service` in `service.go`, `Datastore` in `datastore.go`
- **Service implementations**: `server/service/` — business logic, auth checks
- **Datastore (MySQL)**: `server/datastore/mysql/` — SQL queries, migrations
- **Enterprise features**: `ee/server/service/` — wraps core service with license checks
- **MDM**: `server/mdm/` — Apple, Microsoft, Android device management
- **Frontend**: `frontend/pages/` (routes), `frontend/components/` (reusable UI), `frontend/services/` (API client)
- **CLI tools**: `cmd/fleet/` (server), `cmd/fleetctl/` (management CLI), `orbit/` (agent)

### Enterprise vs core
- Core features: no special build tags, available in all deployments
- Enterprise features: in `ee/` directory, license checks at service layer
- Use `//go:build !premium` for core-only features when needed

## Terminology

The following terms were recently renamed. Use the new terms in conversation and new code, but don't rename existing variables or API parameters without guidance:
- **"Teams" → "Fleets"** — the concept of grouping hosts. Legacy code still uses `team_id`, `teams` table, etc.
- **"Queries" → "Reports"** — what was formerly a "query" in the product is now a "report." The word "query" now refers solely to a SQL query, which is one aspect of a report.

## Fleet-specific patterns

### Go backend
- **Error wrapping**: `ctxerr.Wrap(ctx, err, "description")` — never pkg/errors
- **Request/Response**: lowercase struct types, `Err error` field, `Error()` method returning `r.Err`
- **Endpoint registration**: `ue.POST("/api/_version_/fleet/resource", fn, reqType{})`
- **Authorization**: `svc.authz.Authorize(ctx, entity, fleet.ActionX)` at start of service methods
- **Logging**: slog with `DebugContext/InfoContext/WarnContext/ErrorContext` — never bare slog.Debug/Info/Warn/Error
- **Pointers**: Use Go 1.26 `new(expression)` for pointer values (e.g., `new("value")`, `new(true)`, `new(42)`). Do NOT use the legacy `server/ptr` package in new code — it exists throughout the codebase but is superseded by `new(expr)`.
- **Reference example**: `server/service/vulnerabilities.go`

## Before writing a fix

- Identify WHERE in the request lifecycle the problem manifests (creation vs team-addition vs sync vs query). Fix it there, not at the reproduction step.
- Read the surrounding 100 lines. If similar checks exist nearby, follow their pattern exactly.
- If an endpoint has zero DB interaction, that's intentional. Adding DB calls needs justification.
- Cover ALL entry points for the same operation (single add, batch/GitOps, etc.).
- For declarative/batch endpoints, validate within the incoming payload, not against the DB.
- When checking for duplicates, exclude the current entity to avoid false conflicts on upserts.
- Run `go test ./server/service/` after adding new datastore interface methods — uninitialized mocks crash other tests.

## Development commands

Check the `Makefile` for the full list of available targets. Key ones below.

### Building and running
```bash
make build          # Build fleet + fleetctl
make serve          # Start dev server (or: make up)
make generate-dev   # Webpack watch mode for frontend dev
make deps           # Install dependencies
```

### Testing
```bash
go test ./server/fleet/...                                          # Quick (no external deps)
MYSQL_TEST=1 go test ./server/datastore/mysql/...                   # MySQL integration
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...              # Service integration
MYSQL_TEST=1 go test -run TestFunctionName ./server/datastore/mysql/... # Specific test
yarn test                                                            # Frontend Jest tests
```

### Linting
```bash
make lint-go-incremental  # Go — ONLY changes since branching from main (use after editing)
make lint-go              # Go — full (use before committing)
make lint-js              # JS/TS linters
```

### Database
```bash
make migration name=CamelCaseName   # Create new migration
make db-reset                       # Reset dev database
```

### CI test bundles
| Bundle | Packages | Env vars |
|--------|----------|----------|
| `fast` | No external deps | none |
| `mysql` | `server/datastore/mysql/...` | `MYSQL_TEST=1` |
| `service` | `server/service/` (unit) | `MYSQL_TEST=1 REDIS_TEST=1` |
| `integration-core` | `server/service/integration_*_test.go` | `MYSQL_TEST=1 REDIS_TEST=1` |
| `integration-enterprise` | `ee/server/service/integration_*_test.go` | `MYSQL_TEST=1 REDIS_TEST=1` |
| `integration-mdm` | MDM integration tests | `MYSQL_TEST=1 REDIS_TEST=1` |
| `fleetctl` | `cmd/fleetctl/...` | varies |
| `vuln` | `server/vulnerabilities/...` | varies |
| `main` | Everything else | varies |

## Skills and agents

Type `/` to see available skills. Key ones: `/test`, `/lint`, `/review-pr`, `/fix-ci`, `/spec-story`, `/new-endpoint`, `/new-migration`, `/bump-migration`, `/project`, `/fleet-gitops`, `/find-related-tests`.

Agents: **go-reviewer** (proactive after Go edits), **frontend-reviewer** (proactive after TS edits), **fleet-security-auditor** (on-demand for auth/MDM/security).

## Documentation

All Fleet documentation lives in this repo. Check these sources before searching the web:

- **`docs/`** — User-facing docs: feature guides, REST API reference, configuration, deployment, contributing
- **`handbook/`** — Internal procedures: engineering practices, company policies, product design
- **`articles/`** — Blog posts and tutorials

## Other references

- Linter config: `.golangci.yml`
- Activity types: `docs/Contributing/reference/audit-logs.md`
- Claude Code setup: `.claude/README.md`
