# Fleet

Fleet is an open-source device management platform (MDM + osquery). Large Go monorepo with a React/TypeScript frontend.

## Tech Stack

- **Backend:** Go 1.25 — gorilla/mux routing, sqlx + goqu for MySQL, redigo for Redis, cobra CLI
- **Frontend:** React 18 + TypeScript — webpack, Jest, Storybook
- **Enterprise code:** Lives in `ee/` (licensed separately)

## Project Structure

```
cmd/fleet/       - Fleet server binary (serve, prepare db, etc.)
cmd/fleetctl/    - CLI tool for managing Fleet
server/          - Backend: service layer, datastore, API handlers
server/datastore/mysql/          - MySQL datastore implementation
server/datastore/mysql/migrations/ - DB migrations (tables/ and data/)
server/service/  - Business logic service layer
server/fleet/    - Core types and interfaces
ee/              - Enterprise edition features
frontend/        - React web UI
orbit/           - Orbit agent (runs on endpoints)
tools/           - Utility scripts
```

## Running Tests

```bash
# Quick Go tests (no external deps)
go test ./server/fleet/...

# Integration tests (need MySQL and/or Redis running)
MYSQL_TEST=1 go test ./server/datastore/mysql/...
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...

# Run a specific test
MYSQL_TEST=1 go test -run TestFunctionName ./server/datastore/mysql/...

# Makefile targets
make test          # lint + test-go + test-js
make test-go       # Go tests (uses CI_TEST_PKG for bundles)
make test-js       # Jest frontend tests
make lint-go       # golangci-lint
```

## Testing Patterns

- Uses `testify` (assert/require) everywhere
- MySQL integration tests use `CreateMySQLDS(t)` helper
- Tests run against real MySQL databases created dynamically
- SQL mocking via `go-sqlmock` for unit tests without DB
- Always `defer ds.Close()` after creating a test datastore

## Database Migrations

- Location: `server/datastore/mysql/migrations/tables/` and `data/`
- Create new: `make migration name=DescriptiveName`
- Uses custom goose wrapper at `server/goose/`
- Each migration should have a corresponding `_test.go`
- Migration tests use `applyUpToPrev(t)` and `applyNext(t, db)` helpers

## Code Style

- **Go:** golangci-lint with strict rules (see `.golangci.yml`)
  - Use `ctxerr` package for errors, NOT `github.com/pkg/errors`
  - Use structured logging (slog), NOT print/println
  - Use context-aware slog methods
- **Frontend:** ESLint + Prettier defaults
- **Commits:** Capitalized, imperative mood, issue number in parens: `Add feature X (#12345)`

## Key Patterns

- Errors: wrap with `ctxerr.Wrap(ctx, err, "message")` or `ctxerr.New(ctx, "message")`
- Logging: `level.Info(logger).Log("msg", "description", "key", value)` or slog
- Auth: JWT-based (`golang-jwt/jwt/v4`)
- DB queries: sqlx for simple queries, goqu query builder for complex ones
- API: REST via gorilla/mux, responses via `fleet.Response` types
