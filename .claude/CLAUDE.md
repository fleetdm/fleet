# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About Fleet

Fleet is an open-source platform for IT and security teams: device management (MDM), vulnerability reporting, osquery fleet management, and security monitoring. Go backend, React/TypeScript frontend, manages thousands of devices across macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS.

## Architecture

### Backend Request Flow
HTTP request → `server/service/handler.go` routes → endpoint function (decode request) → service method (auth + business logic) → datastore method (SQL) → response struct

### Key Layers
- **Types & interfaces**: `server/fleet/` — `Service` in `service.go`, `Datastore` in `datastore.go`
- **Service implementations**: `server/service/` — business logic, auth checks
- **Datastore (MySQL)**: `server/datastore/mysql/` — SQL queries, migrations
- **Enterprise features**: `ee/server/service/` — wraps core service with license checks
- **MDM**: `server/mdm/` — Apple, Microsoft, Android device management
- **Frontend**: `frontend/pages/` (routes), `frontend/components/` (reusable UI), `frontend/services/` (API client)
- **CLI tools**: `cmd/fleet/` (server), `cmd/fleetctl/` (management CLI), `orbit/` (agent)

### Enterprise vs Core
- Core features: no special build tags, available in all deployments
- Enterprise features: in `ee/` directory, license checks at service layer
- Use `//go:build !premium` for core-only features when needed

## Fleet-Specific Patterns

### Go Backend
- **Error wrapping**: `ctxerr.Wrap(ctx, err, "description")` — never pkg/errors
- **Request/Response**: lowercase struct types, `Err error` field, `Error()` method returning `r.Err`
- **Endpoint registration**: `ue.POST("/api/_version_/fleet/resource", fn, reqType{})`
- **Authorization**: `svc.authz.Authorize(ctx, entity, fleet.ActionX)` at start of service methods
- **Logging**: slog with `DebugContext/InfoContext/WarnContext/ErrorContext` — never bare slog.Debug/Info/Warn/Error
- **Pointer utilities**: `server/ptr` — `ptr.String()`, `ptr.Uint()`, `ptr.Bool()`, `ptr.ValOrZero()`
- **Reference example**: `server/service/vulnerabilities.go`

### Go Code Style
- Prefer `map[T]struct{}` over `map[T]bool` for sets
- Convert map keys to slice: `slices.Collect(maps.Keys(m))`
- Avoid `time.Sleep` in tests — use `testing/synctest`, polling helpers, channels, or `require.Eventually`
- Use `require`/`assert` from `github.com/stretchr/testify`
- Use `t.Context()` instead of `context.Background()` in tests
- Use `any` instead of `interface{}`

### Frontend
- **Component structure**: `.tsx` + `_styles.scss` + `.tests.tsx` + `index.ts`
- **Data fetching**: React Query (`useQuery` with `[key, dep]` and `enabled`)
- **API calls**: `sendRequest(method, path, body?, params?)` from `frontend/services/`
- **Styling**: SCSS with BEM — `const baseClass = "component-name"`
- **Interfaces**: `frontend/interfaces/` with `I` prefix (IHost, IUser)
- **Component generator**: `./frontend/components/generate -n PascalName -p optional/path`

## Before Writing a Fix

- Identify WHERE in the request lifecycle the problem manifests (creation vs team-addition vs sync vs query). Fix it there, not at the reproduction step.
- Read the surrounding 100 lines. If similar checks exist nearby, follow their pattern exactly.
- If an endpoint has zero DB interaction, that's intentional. Adding DB calls needs justification.
- Cover ALL entry points for the same operation (single add, batch/GitOps, etc.).
- For declarative/batch endpoints, validate within the incoming payload, not against the DB.
- When checking for duplicates, exclude the current entity to avoid false conflicts on upserts.
- Run `go test ./server/service/` after adding new datastore interface methods — uninitialized mocks crash other tests.
- Use existing utilities (`ptr.ValOrZero`, etc.) and follow parameter style conventions.

## Development Commands

### Building & Running
```bash
make build          # Build fleet + fleetctl
make fleet          # Build fleet server only
make fleetctl       # Build fleetctl CLI only
make serve          # Start dev server (or: make up)
make generate       # Generate frontend assets + Go code
make generate-dev   # Webpack watch mode for frontend dev
make deps           # Install dependencies
```

### Testing
```bash
# Go tests
go test ./server/fleet/...                                          # Quick (no external deps)
MYSQL_TEST=1 go test ./server/datastore/mysql/...                   # MySQL integration
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...              # Service integration
MYSQL_TEST=1 go test -run TestFunctionName ./server/datastore/mysql/... # Specific test
FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 MYSQL_TEST=1 REDIS_TEST=1 go test -run TestName ./server/service/... # Quiet mode

# Frontend
yarn test                   # Jest tests
yarn lint                   # ESLint
npx prettier --check frontend/  # Formatting check

# Full suite
make test           # Lint + Go + JS
make lint-go        # Go linters only
make lint-js        # JS/TS linters only
```

### Database
```bash
make migration name=CamelCaseName   # Create new migration
make db-reset                       # Reset dev database
make db-backup                      # Backup dev database
make db-restore                     # Restore from backup
```

### E2E
```bash
make e2e-reset-db       # Reset E2E database
make e2e-serve-free     # Start server (free edition)
make e2e-serve-premium  # Start server (premium edition)
```

### CI Test Bundles
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

## Common Workflows

1. **Starting development**: `make deps && make generate-dev && make serve`
2. **Running tests**: `make test` (full) or `make run-go-tests PKG_TO_TEST="specific/package"`
3. **Database changes**: `make migration name=YourChange` then `make db-reset`
4. **Frontend changes**: `make generate-dev` for live reloading
5. **Adding features**: service → datastore → API → frontend pattern

## Scale & Performance

- Deployments manage thousands of hosts — queries must be efficient at scale
- Multi-tenant architecture with team-based permissions
- Consider impact of real-time features and WebSocket connections
- osquery integration requires careful resource management

## Skills & Agents

### Skills (type `/` to invoke)
- `/review-pr <PR#>` — Review a pull request
- `/fix-ci <run-url>` — Diagnose and fix failing CI tests
- `/test [filter]` — Run tests related to recent changes
- `/find-related-tests` — Find test files for changes
- `/fleet-gitops` — Help with GitOps config files
- `/project <name>` — Load workstream context
- `/new-endpoint` — Scaffold a new API endpoint
- `/new-migration` — Create a database migration
- `/spec-story <issue#>` — Break down a story into implementable sub-issues with technical specs
- `/lint [go|frontend]` — Run linters on recent changes
- `/update-data-dictionary` — Sync DATA-DICTIONARY.md with recent migrations

### Agents (invoked automatically or by name)
- **go-reviewer** — Go changes: bugs, conventions, security (proactive)
- **frontend-reviewer** — React/TypeScript: conventions, type safety (proactive)
- **fleet-security-auditor** — MDM, auth, osquery, device management security

## Documentation

All Fleet documentation lives in this repo. When answering questions about Fleet's features, APIs, deployment, or workflows, check these sources before searching the web:

- **`docs/`** — User-facing documentation
  - `docs/01-Using-Fleet/` — Feature guides
  - `docs/REST API/` — REST API reference
  - `docs/Configuration/` — Server and agent configuration
  - `docs/Deploy/` — Deployment guides
  - `docs/Contributing/` — Development guides, API conventions, and audit log reference
- **`handbook/`** — Internal procedures and workflows
  - `handbook/engineering/` — Engineering practices and rituals
  - `handbook/company/` — Company-wide policies including writing style guide
  - `handbook/product-design/` — Product and design processes
- **`articles/`** — Blog posts and tutorials (438 articles)
- **Root files** — `README.md`, `CHANGELOG.md`, `SECURITY.md`, `DATA-DICTIONARY.md`

## Other references

- Linter config: `.golangci.yml`
- Activity types: `docs/Contributing/reference/audit-logs.md`
- Claude Code setup: `.claude/README.md`
