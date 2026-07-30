---
paths:
  - "server/**/*.go"
  - "cmd/**/*.go"
  - "orbit/**/*.go"
  - "ee/**/*.go"
  - "pkg/**/*.go"
  - "tools/**/*.go"
  - "client/**/*.go"
  - "test/**/*.go"
---

# Fleet Go Backend Conventions

## Error Handling
- Wrap errors with `ctxerr.Wrap(ctx, err, "description")` — never `pkg/errors` or `fmt.Errorf` with `%w`
- For error messages without wrapping, use `errors.New("msg")` not `fmt.Errorf("msg")` (the linter catches this)
- Banned imports: `github.com/pkg/errors`, `github.com/valyala/fastjson`, `github.com/valyala/fasttemplate`
- Use the right error type for the right situation:
  - `fleet.NewInvalidArgumentError(field, reason)` — input validation (422). Accumulate with `.Append(field, reason)`, check `.HasErrors()`
  - `&fleet.BadRequestError{Message: "..."}` — malformed request (400)
  - `fleet.NewAuthFailedError()` / `fleet.NewAuthRequiredError()` — auth failures (401)
  - `fleet.NewPermissionError(msg)` — authorized but insufficient role (403)
  - Implement `IsNotFound() bool` interface — resource not found. Check with `fleet.IsNotFound(err)`
  - `&fleet.ConflictError{Message: "..."}` — duplicate/conflict (409)
- Check error types with: `fleet.IsNotFound(err)`, `fleet.IsAlreadyExists(err)`

## Input Validation
- Validate in service methods, not in endpoint functions
- Accumulate all errors before returning:
  ```go
  invalid := fleet.NewInvalidArgumentError("name", "cannot be empty")
  if badCondition {
      invalid.Append("email", "must be valid")
  }
  if invalid.HasErrors() {
      return invalid
  }
  ```

## Service Methods
- Signature: `func (svc *Service) MethodName(ctx context.Context, ...) (..., error)`
- Start with authorization: `svc.authz.Authorize(ctx, &fleet.Entity{}, fleet.ActionX)`
- For entity-specific auth, double-authorize: generic check first, load entity, then team-scoped check:
  ```go
  if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionRead); err != nil { return nil, err }
  host, err := svc.ds.Host(ctx, hostID)
  if err != nil { return nil, ctxerr.Wrap(ctx, err, "get host") }
  if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil { return nil, err }
  ```
- Return errors via ctxerr wrapping

## Viewer Context
- Get current user: `vc, ok := viewer.FromContext(ctx)` — NEVER trust user identity from request body
- Helpers: `vc.UserID()`, `vc.Email()`, `vc.IsLoggedIn()`, `vc.CanPerformActions()`
- System operations: `viewer.NewSystemContext(ctx)` for admin-level automated actions

## Pagination
- Use `fleet.ListOptions` for all list endpoints (Page, PerPage, OrderKey, OrderDirection, MatchQuery, After)
- Return `*fleet.PaginationMetadata` when `IncludeMetadata` is true
- Cursor pagination: check `ListOptions.UsesCursorPagination()`

## Request/Response Pattern
- Request structs: lowercase type, json/url tags: `type listEntitiesRequest struct`
- Response structs: include `Err error` field and `func (r xResponse) Error() error { return r.Err }`
- Endpoint functions: `func xEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error)`
- Errors go in the response body: `return xResponse{Err: err}, nil`

## Logging
- Use slog with context: `logger.InfoContext(ctx, "message", "key", value)`
- NEVER use bare `slog.Debug`, `slog.Info`, `slog.Warn`, `slog.Error` — the `forbidigo` linter rejects these
- NEVER use `print()` or `println()` — use structured logging

## Imports & Utilities
- Internal packages: `github.com/fleetdm/fleet/v4/server/` prefix
- **HTTP clients**: Use `fleethttp.NewClient()` — never `http.Client{}` or `new(http.Client)` directly (custom linter rule)
- **Pointers**: Use Go 1.26 `new(expression)` (e.g., `new("value")`, `new(true)`, `new(42)`) — not `server/ptr`'s deprecated constructors. Non-deprecated helpers are still fine: `ptr.ValOrZero` (deref-or-zero), `ptr.Equal` (nil-safe equality), `ptr.UintOrNilIfZero` (nil for `0`).
- **Random numbers**: use `math/rand/v2` instead of `math/rand`
- Sets: use `map[T]struct{}`, convert to slice with `slices.Collect(maps.Keys(m))`
- Flexible JSON: use `json.RawMessage` for configs stored as JSON blobs

## Context Utilities
- `ctxdb.RequirePrimary(ctx, true)` — force reads on primary DB (use before read-then-write)
- `ctxdb.BypassCachedMysql(ctx, true)` — disable MySQL cache layer
- `ctxerr.Wrap(ctx, err, "msg")` — ALWAYS use for error wrapping

## Testing
- Use `require` and `assert` from `github.com/stretchr/testify`
- Mock invocation tracking: check `ds.{FuncName}FuncInvoked` bool (auto-set by generated mocks)
- Run `go test ./server/service/` after adding new datastore interface methods — uninitialized mocks crash other tests
- Integration tests need `MYSQL_TEST=1 REDIS_TEST=1`
- Use `t.Context()` instead of `context.Background()`

## Bounded contexts

Some domains use a self-contained bounded context pattern instead of the traditional `fleet/` → `service/` → `datastore/` layers:
- `server/activity/` — internal types, mysql, service, API, and bootstrap in one directory
- `server/mdm/` — similar self-contained structure for MDM

When working in these directories, follow the local patterns (internal packages, local types) rather than the top-level Fleet architecture.

## Linting
- Follow `.golangci.yml` — enabled linters: depguard, forbidigo, gosec, gocritic, revive, errcheck, staticcheck
- After editing: `make lint-go-incremental` (only checks changes since branching from main)
- Before committing: `make lint-go` (full lint)
