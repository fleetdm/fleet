# ADR-0008: Migrate from go-kit/log to slog

## Status

Proposed

## Date

2026-01-24

## Context

Fleet currently uses [go-kit/log](https://github.com/go-kit/log) for structured logging throughout the codebase. While go-kit/log has served us well, it has a fundamental limitation: **it lacks native context support.** The `Logger.Log(keyvals...)` interface doesn't accept a `context.Context` parameter.

This limitation became apparent when implementing [ADR-0005: Standardize on OpenTelemetry for observability](0005-opentelemetry-standardization.md). To correlate logs with distributed traces, we need to inject `trace_id` and `span_id` from the OpenTelemetry span context into log entries. With go-kit/log, this requires fragile workarounds like pointer-based context tracking or manually recreating loggers when spans change.

The standard solution is context-aware logging, which Go 1.21 introduced with the `log/slog` package.

### Industry adoption

slog has seen rapid adoption since its introduction in Go 1.21:

- **[Teleport](https://github.com/gravitational/teleport/pull/34173/changes)**: Migrating from logrus to slog, citing that "as part of the standard library, slog is likely to enjoy long-term support" and that logrus is "no longer actively developed"

- **[Kubernetes klog](https://github.com/kubernetes/klog)**: Added slog interoperability in v2.110.1, allowing applications to route log output between klog and slog handlers. Their [Contextual Logging KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/3077-contextual-logging/README.md#integration-with-logslog) explicitly references slog integration

- **[go-kit/log interop proposal](https://github.com/golang/go/issues/65613)**: A proposal was made for slog to implement go-kit/log's Logger interface, demonstrating demand from projects wanting to migrate away from go-kit while maintaining compatibility with dependencies

- **Go community**: The [awesome-slog](https://github.com/go-slog/awesome-slog) repository tracks the growing ecosystem of slog handlers and integrations

The Go team noted that "structured logging has consistently ranked high in Go's annual survey" and slog is "one of the largest additions to the standard library since Go 1 was released in 2012."

### Current state

- ~320 files import `github.com/go-kit/log`
- Logging is used in services, cron jobs, workers, MDM components, and HTTP middleware
- Some third-party dependencies (nanodep, nanomdm) also use go-kit/log patterns

## Decision

Fleet will migrate from go-kit/log to Go's standard library `log/slog` package. This aligns with Fleet's [🟢 Results](https://fleetdm.com/handbook/company#results) value: "Keep it simple. Choose boring solutions. Reuse systems."

We will:

1. **Adopt slog** as the standard logging interface for all new code
2. **Use otelslog bridge** for automatic OpenTelemetry trace correlation
3. **Migrate incrementally** using an adapter layer during transition
4. **Update existing code** package-by-package over multiple releases
5. **Deprecate go-kit/log usage** once migration is complete

### Implementation plan

#### Phase 1: Foundation

1. Add slog and OpenTelemetry bridge dependencies:
   - `log/slog` (standard library)
   - `go.opentelemetry.io/contrib/bridges/otelslog` for trace correlation

2. Create `server/platform/logging` package with:
   - Default slog configuration (JSON handler for production, text for development)
   - OpenTelemetry handler wrapper for automatic trace_id/span_id injection
   - Adapter that implements `kitlog.Logger` interface using slog (for gradual migration)

3. Update application initialization in `cmd/fleet/serve.go` to configure slog as default logger

4. Add linting rule to enforce context-aware logging methods (`slog.InfoContext`, etc.) and flag non-context methods (`slog.Info`, etc.) to ensure trace correlation is not bypassed

#### Phase 2: Incremental migration

Migrate packages in order of dependency (leaf packages first):

1. **server/contexts/logging** - Update LoggingContext to use slog
2. **server/service/schedule** - Cron job scheduling
3. **server/worker** - Background job processing
4. **server/service** - Core service layer (largest effort)
5. **server/datastore** - Database layer
6. **cmd/fleet** - Application entry points and cron job definitions

For each package:
- Replace `kitlog.Logger` with `*slog.Logger`
- Replace `logger.Log("key", "value")` with `logger.InfoContext(ctx, "message", "key", "value")`
- Use appropriate log levels (`Debug`, `Info`, `Warn`, `Error`)

#### Phase 3: Cleanup

1. Remove go-kit/log adapter once all code is migrated
2. Remove go-kit/log dependency
3. Update documentation and examples

### Code changes example

Before (go-kit/log with logger parameter):
```go
type Service struct {
    logger kitlog.Logger  // must be passed to every struct
}

func (s *Service) DoSomething(ctx context.Context) error {
    level.Info(s.logger).Log("msg", "starting operation", "user_id", userID)
    level.Error(s.logger).Log("msg", "operation failed", "err", err)
}
```

After (slog):
```go
type Service struct {
    logger *slog.Logger
}

func (s *Service) DoSomething(ctx context.Context) error {
    s.logger.InfoContext(ctx, "starting operation", "user_id", userID)
    s.logger.ErrorContext(ctx, "operation failed", "err", err)
}
```

With OpenTelemetry configured, logs automatically include `trace_id` and `span_id` when a span is active in the context.

## Consequences

### Positive

- **Native context support**: `slog.InfoContext(ctx, ...)` passes context to every log call, enabling automatic trace correlation without workarounds
- **Standard library with zero dependencies**: No external packages required, always compatible with Go versions, stable API guaranteed by Go's compatibility promise
- **OpenTelemetry integration**: Official `otelslog` bridge provides automatic trace_id/span_id injection
- **Built-in testing support**: The [`testing/slogtest`](https://pkg.go.dev/testing/slogtest) package provides utilities for verifying handler implementations; log messages can serve as test assertions for code paths and variable values
- **Leveled logging built-in**: Debug, Info, Warn, Error levels are first-class citizens, unlike go-kit/log which requires the separate `level` package
- **JSON output built-in**: `slog.JSONHandler` provides production-ready JSON logging without additional configuration; go-kit/log requires custom encoders
- **Handler extensibility**: Easy to swap or chain handlers for different output formats, filtering, or destinations (e.g., [slog-multi](https://github.com/samber/slog-multi) for fan-out)
- **Attribute type safety**: `slog.Attr` provides typed attributes, reducing runtime errors from mismatched key-value pairs
- **Reduced onboarding friction**: New team members familiar with Go's standard library can contribute immediately without learning go-kit patterns

### Negative

- **Migration effort**: ~320 files need updates; this is a significant undertaking
- **Learning curve**: Team members need to learn slog patterns
- **Third-party dependencies**: Some forked packages (nanodep, nanomdm) use go-kit patterns and may need updates
- **Temporary complexity**: During transition, codebase will have both logging styles
- **Breaking changes**: Logger interface changes may affect any code that accepts logger parameters
- **Log format changes**: See below

### Log output format changes

slog's JSONHandler defaults differ from Fleet's current log format. To minimize disruption for customers parsing Fleet logs, we will configure slog via [`ReplaceAttr`](https://pkg.go.dev/log/slog#HandlerOptions) to maintain the current format:

| Field            | Current format  | slog default    | Our configuration           |
|------------------|-----------------|-----------------|-----------------------------|
| Timestamp key    | `ts`            | `time`          | `ts` (preserved)            |
| Timestamp format | RFC3339         | RFC3339Nano     | RFC3339 (preserved)         |
| Level case       | `info`, `debug` | `INFO`, `DEBUG` | `info`, `debug` (preserved) |

This ensures backward compatibility; any existing log parsing tools will continue to work without modification.

### Future considerations

- **Logger-in-context pattern**: Consider implementing `WithLogger(ctx, logger)` and `FromContext(ctx)` helpers to store/retrieve the logger from context, eliminating the need to pass logger as a parameter. This is a common pattern (used by [go-logr/logr](https://pkg.go.dev/github.com/go-logr/logr), [slog-context](https://github.com/veqryn/slog-context)) though deliberately not included in slog's standard library
- Evaluate slog handlers for specific backends (e.g., GCP Cloud Logging)
- Monitor slog ecosystem for new handlers and best practices

## Alternatives considered

### Keep go-kit/log with a wrapper holding a context pointer

- **Pros**: No migration effort, works with existing code
- **Cons**: Fragile pointer-based pattern, not industry standard, requires manual intervention at every span creation, easy to mess up
- **Rejected because**: The workarounds are error-prone and don't follow Go idioms

### Migrate to zerolog or zap

- **Pros**: Mature libraries with proven performance, context support available via wrappers
- **Cons**: External dependency, different API to learn, less ecosystem momentum than slog
- **Rejected because**: slog is now standard library with official OpenTelemetry support; adding another third-party logging dependency goes against Fleet's simplification goals

### Use slog with go-kit adapter permanently

- **Pros**: Minimal code changes, existing code continues to work
- **Cons**: Still requires adapter maintenance, doesn't get full slog benefits, context still not native
- **Rejected because**: Adapter layer should be transitional, not permanent; full migration provides cleaner codebase

## References

### Official documentation
- [Structured Logging with slog - The Go Programming Language](https://go.dev/blog/slog)
- [slog package - Go standard library](https://pkg.go.dev/log/slog)
- [testing/slogtest - Handler testing utilities](https://pkg.go.dev/testing/slogtest)

### OpenTelemetry integration
- [otelslog - Official OpenTelemetry slog bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog)
- [slog-otel - Alternative handler for trace correlation](https://github.com/remychantenay/slog-otel)

### Guides and tutorials
- [Logging in Go with Slog: The Ultimate Guide - Better Stack](https://betterstack.com/community/guides/logging/logging-in-go/)
- [Contextual Logging in Go with Slog - Better Stack](https://betterstack.com/community/guides/logging/golang-contextual-logging/)
- [Deep Dive and Migration Guide to Go 1.21+'s slog - Leapcell](https://leapcell.io/blog/deep-dive-and-migration-guide-to-go-1-21-s-structured-logging-with-slog)

### Fleet-specific
- [ADR-0005: Standardize on OpenTelemetry for observability](0005-opentelemetry-standardization.md)
- [GitHub Issue #38607: Link OTEL traces with logs](https://github.com/fleetdm/fleet/issues/38607)
