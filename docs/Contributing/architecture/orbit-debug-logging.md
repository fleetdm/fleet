# Orbit debug logging

This document describes how debug logging works in the Fleet orbit agent today, and the design for a new runtime-togglable debug logging feature that can be controlled per-team (via agent options) and per-host (via a new API endpoint with automatic expiry).

- [Existing architecture](#existing-architecture)
  - [Entry points](#entry-points)
  - [Logger setup](#logger-setup)
  - [Propagation to osqueryd](#propagation-to-osqueryd)
  - [Signal handling](#signal-handling)
  - [Limitations](#limitations)
- [New feature: runtime-togglable debug logging](#new-feature-runtime-togglable-debug-logging)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
  - [Data model](#data-model)
  - [API surface](#api-surface)
  - [Data flow](#data-flow)
  - [Precedence rules](#precedence-rules)
  - [Auto-expiry](#auto-expiry)
  - [Osquery flag synchronization](#osquery-flag-synchronization)
- [Implementation plan](#implementation-plan)
- [Open questions](#open-questions)

## Existing architecture

### Entry points

Today, orbit debug logging can only be enabled **at process startup**, via two equivalent mechanisms:

| Mechanism | Location | Notes |
|-----------|----------|-------|
| `--debug` CLI flag | `orbit/cmd/orbit/orbit.go:178-181` | Attached to the main `orbit` command |
| `ORBIT_DEBUG` env var | Same declaration | `altsrc` env-var alias for the flag |
| `--debug` on `orbit shell` | `orbit/cmd/orbit/shell.go:37-41` | Same mechanism, different subcommand |

Operators flip the flag by editing the orbit launchd/systemd/service unit and restarting the service.

### Logger setup

Orbit uses [rs/zerolog](https://github.com/rs/zerolog). The level is set **once** in `orbitAction` at `orbit/cmd/orbit/orbit.go:331-335`:

```go
zerolog.SetGlobalLevel(zerolog.InfoLevel)
if c.Bool("debug") {
    zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
```

Log output is a multi-writer composed at `orbit/cmd/orbit/orbit.go:296-329`:

- A rotating file writer (`lumberjack`) at `{root-dir}/orbit.log`.
- `os.Stderr` for console output when running interactively.

`SetGlobalLevel` is never called again after startup — confirmed by grepping the orbit tree. This is the core reason debug logging can't be toggled at runtime today.

### Propagation to osqueryd

When orbit starts osqueryd as a child process, it appends verbose flags **only if** `--debug` was set. From `orbit/cmd/orbit/orbit.go:1349-1353`:

```go
if c.Bool("debug") {
    options = append(options,
        osquery.WithFlags([]string{"--verbose", "--tls_dump"}),
    )
}
```

These flags are baked into the osqueryd invocation and cannot change without restarting osqueryd. Separately, the server can push arbitrary osquery command-line flags via agent options — see [osquery flag synchronization](#osquery-flag-synchronization) below.

### Signal handling

Orbit installs one signal handler on Unix platforms: `sigusrListener` at `orbit/cmd/orbit/signal_unix.go:28-38`, wired up at `orbit/cmd/orbit/orbit.go:1547`. It responds to `SIGUSR1` by writing pprof profiles (CPU, heap, goroutine) to `{root-dir}/profiles/`. **It does not touch the log level.**

There is no `SIGHUP` handler, no config-file watcher, and no IPC/HTTP control endpoint inside orbit.

### Limitations

1. **Restart required.** Enabling or disabling debug mode requires restarting the orbit service, which is disruptive and often requires remote-access tooling to perform at a customer site.
2. **All-or-nothing.** Debug mode applies to every log statement in orbit. There is no category/component filtering.
3. **Osqueryd verbosity coupled to orbit `--debug`.** The only way to get osqueryd `--verbose` via the startup flag path is to also put orbit into debug mode (separate from the agent-options `command_line_flags` path, which is already runtime-configurable but team-wide).

## New feature: runtime-togglable debug logging

### Goals

- Allow admins to toggle orbit debug logging for an entire team (default OFF).
- Allow admins to toggle orbit debug logging for a single host, with a time-boxed auto-expiry so it cannot be left on indefinitely (common support-engineering workflow).
- When debug is enabled, osqueryd should also become verbose — but only if the resulting flags actually differ from what osqueryd is running with, to avoid unnecessary osqueryd restarts.
- No orbit restart required to flip the switch either way.

### Non-goals

- Per-component or per-package log filtering.
- Modifying what is logged at debug level (a security review of debug-level output is deferred — see [Open questions](#open-questions)).
- Dynamic verbosity for the Fleet server or Fleet Desktop processes.

### Data model

#### Team tier — agent options

Extend the `AgentOptions` struct in `server/fleet/agent_options.go:18-31` with a new top-level `orbit` block:

```json
{
  "config": { ... },
  "overrides": { ... },
  "command_line_flags": { ... },
  "extensions": { ... },
  "orbit": {
    "debug_logging": true
  }
}
```

Rationale for a new `orbit` key rather than reusing `command_line_flags`: the osquery flags block is strictly validated against the osquery flags schema in `agent_options_generated.go`. Orbit-level concerns don't belong in that namespace. A dedicated `orbit` block also gives us somewhere to grow.

Concretely, add to `server/fleet/agent_options.go`:

```go
type AgentOptions struct {
    // ...existing fields...
    Orbit *OrbitAgentOptions `json:"orbit,omitempty"`
}

type OrbitAgentOptions struct {
    DebugLogging bool `json:"debug_logging,omitempty"`
}
```

Validated by `ValidateJSONAgentOptions` at `server/fleet/agent_options.go:72` — the existing `JSONStrictDecode` will automatically reject unknown subkeys under `orbit`.

#### Host tier — new column

Add a timestamp column to `hosts`:

```sql
ALTER TABLE hosts
  ADD COLUMN orbit_debug_until TIMESTAMP NULL DEFAULT NULL;
```

- `NULL` → no host-level override; team default applies.
- Non-NULL, future → debug forced ON regardless of team setting.
- Non-NULL, past → lapsed; treated as NULL (a cleanup job can null it out lazily).

Migration filename follows the `YYYYMMDDhhmmss_DescriptiveName.go` convention (see `server/datastore/mysql/migrations/tables/` for the current format, e.g. `20260410173222_AddHTTPETagToSoftwareInstallers.go`).

The `Host` struct in `server/fleet/hosts.go` gains a matching field:

```go
OrbitDebugUntil *time.Time `json:"orbit_debug_until,omitempty" db:"orbit_debug_until"`
```

#### Wire protocol

Extend `OrbitConfig` in `server/fleet/orbit.go:60-70`:

```go
type OrbitConfig struct {
    // ...existing fields...
    DebugLogging *bool `json:"debug_logging,omitempty"`
}
```

Pointer type so "unset" is distinct from "explicitly false" — lets older servers and newer orbits coexist without behavior change.

### API surface

#### Team-level

No new endpoints. Existing agent-options flows cover this:

- `PATCH /api/v1/fleet/config` (global, no-team)
- `PATCH /api/v1/fleet/teams/:id/agent_options`
- `fleetctl apply -f team.yml`
- UI: Settings → Organization settings / Team settings → Agent options

#### Host-level — new

```
POST   /api/v1/fleet/hosts/:id/debug-logging
Body:  { "enabled": true, "duration": "1h" }      # or "expires_at": "<RFC3339>"
Response: { "orbit_debug_until": "2026-04-15T17:00:00Z" }

DELETE /api/v1/fleet/hosts/:id/debug-logging
Response: 204 No Content
```

Endpoint behavior:

- `duration` is a Go-parseable duration string. Default **24h** if omitted. Maximum **7d**; values above cap are rejected with 422 and a `fleet.InvalidArgumentError`.
- Authorization: `admin` or `maintainer` on the host's team (same role gating as host refetch).
- Emits an activity log entry for audit (see [Activities](#activities) below).

Registered via the standard endpoint pattern in `server/service/handler.go`, service method in `server/service/hosts.go`, datastore method in `server/datastore/mysql/hosts.go`.

#### Activities

Two new activity types, modeled on `ActivityTypeEditedAgentOptions` at `server/fleet/activities.go:520-527`:

```go
type ActivityTypeEnabledHostDebugLogging struct {
    HostID          uint      `json:"host_id"`
    HostDisplayName string    `json:"host_display_name"`
    ExpiresAt       time.Time `json:"expires_at"`
}
func (a ActivityTypeEnabledHostDebugLogging) ActivityName() string {
    return "enabled_host_debug_logging"
}

type ActivityTypeDisabledHostDebugLogging struct {
    HostID          uint   `json:"host_id"`
    HostDisplayName string `json:"host_display_name"`
}
func (a ActivityTypeDisabledHostDebugLogging) ActivityName() string {
    return "disabled_host_debug_logging"
}
```

Documented in `docs/Contributing/reference/audit-logs.md` alongside the existing entries.

#### fleetctl

New subcommand under `fleetctl hosts` (not `fleetctl debug`, which targets the Fleet server rather than managed hosts). Slot into `cmd/fleetctl/fleetctl/hosts.go:17-25`:

```
fleetctl hosts debug-logging --host <identifier> --enable [--duration 1h]
fleetctl hosts debug-logging --host <identifier> --disable
```

### Data flow

```
admin flips team agent-options OR posts to /hosts/:id/debug-logging
    │
    ▼
server persists to teams.config / app_config / hosts.orbit_debug_until
    │
    │  (up to 30s later)
    ▼
orbit polls POST /api/fleet/orbit/config      ◄─── client/orbit_client.go:325
    │
    ▼
server's GetOrbitConfig (server/service/orbit.go:355) computes:
    debug := team.agent_options.orbit.debug_logging
    if host.orbit_debug_until != nil && host.orbit_debug_until > now():
        debug = true                    # host override can force ON
    returns OrbitConfig{DebugLogging: &debug, Flags: <merged flags>}
    │
    ▼
orbit's DebugLogRunner receiver:
    if cfg.DebugLogging != nil && zerolog.GlobalLevel() != desiredLevel(cfg):
        zerolog.SetGlobalLevel(desiredLevel(cfg))
        log.Info().Msgf("log level changed to %s by server config", ...)
    │
    ▼
orbit's FlagRunner receiver (existing, orbit/pkg/update/flag_runner.go:47-83):
    diffs server Flags against on-disk osquery.flags
    if different: write file + TriggerOrbitRestart("osquery flags updated")
```

### Precedence rules

1. **Host override is one-way.** A host-level `orbit_debug_until` in the future forces debug ON but cannot force it OFF when the team default is ON. Rationale: ambiguity-free mental model; to turn a specific host off while its team is on, remove it from the team.
2. **Lapsed overrides are ignored.** `orbit_debug_until < now()` is treated as unset.
3. **Startup `--debug` flag is the initial state only.** Once the first `GetOrbitConfig` response arrives, the server's answer takes over. This means `--debug` + server-says-off will flip to off at the first config tick — acceptable for a PoC; revisit if customers object.

### Auto-expiry

Two layers:

- **Enforcement layer** — `GetOrbitConfig` checks `orbit_debug_until > now()` on every call. Hosts naturally drop back to the team default at expiry without any action.
- **Cleanup layer** — a periodic job (fleet cron) sets `orbit_debug_until = NULL` where it's in the past. Keeps the column tidy for debugging and audit queries. Can be added later; not required for correctness.

### Osquery flag synchronization

The existing `FlagRunner` at `orbit/pkg/update/flag_runner.go:47-83` already handles "server pushes new osquery flags → orbit diffs, writes, restarts osqueryd." We lean on this rather than duplicating the logic:

- When `GetOrbitConfig` computes `DebugLogging = true` for a host, it must also **merge** `{"verbose": true, "tls_dump": true}` into the `Flags` blob before returning. The merge goes in the server's response builder so that orbit's existing restart-on-diff logic picks it up automatically.
- When `DebugLogging` flips back to false, the server removes those flags from the blob and `FlagRunner` will restart osqueryd to clear them.
- Critically, the `reflect.DeepEqual` check at `orbit/pkg/update/flag_runner.go:71` ensures **no restart when nothing changes** — if the admin already had `verbose: true` in `command_line_flags`, flipping debug logging on doesn't re-flag or restart osqueryd.

Merge precedence for overlapping keys: admin-specified `command_line_flags` wins over debug-derived flags. Debug mode should never weaken an admin's explicit choice.

## Implementation plan

Order below is a suggested implementation sequence. Each step can be its own commit; the feature is flag-gated by default-off agent option plus default-NULL host column, so partial merges are safe.

### 1. Wire protocol and server-side data model

- Add `DebugLogging *bool` to `fleet.OrbitConfig` — `server/fleet/orbit.go:60-70`.
- Add `Orbit *OrbitAgentOptions` to `fleet.AgentOptions` — `server/fleet/agent_options.go:18-31`.
- Migration: `YYYYMMDDhhmmss_AddHostOrbitDebugUntil.go` adding the `orbit_debug_until` column.
- Add `OrbitDebugUntil *time.Time` to `fleet.Host` — `server/fleet/hosts.go:300` area.
- Datastore methods:
  - `UpdateHostOrbitDebugUntil(ctx, hostID, *time.Time) error` — pattern from `UpdateHostRefetchRequested`.
  - Include `orbit_debug_until` in host SELECT queries where the Host struct is populated.

### 2. Server-side config assembly

- In `GetOrbitConfig` (`server/service/orbit.go:355-645`), compute the effective `DebugLogging` value from `team.Config.AgentOptions.Orbit.DebugLogging` + `host.OrbitDebugUntil`, populate the response.
- Merge `{"verbose": true, "tls_dump": true}` into the returned `Flags` blob when `DebugLogging == true`, respecting admin-specified flags.
- Unit tests in `server/service/orbit_test.go` covering: team-off/host-off, team-on/host-off, team-off/host-on, team-off/host-lapsed, admin `verbose:false` conflict resolution.

### 3. Host-level API endpoint

- Request/response structs in `server/service/hosts.go`.
- Endpoint registration in `server/service/handler.go` — mirror the refetch registration at `server/service/handler.go:466`.
- Service method with authz + duration validation (default 24h, cap 7d).
- Activity log entries — `server/fleet/activities.go` + docs in `docs/Contributing/reference/audit-logs.md`.
- Integration tests under `server/service/integration_core_test.go`.

### 4. Orbit-side receiver

- New file: `orbit/pkg/update/debug_log_runner.go` implementing `fleet.OrbitConfigReceiver`.
  - Compares `cfg.DebugLogging` against `zerolog.GlobalLevel()`.
  - Calls `zerolog.SetGlobalLevel(...)` only when a change is required.
  - Logs the transition at info level.
- Register the receiver in `orbit/cmd/orbit/orbit.go` alongside `FlagRunner`.
- Unit test for the receiver (mock config, assert level changes).

### 5. fleetctl

- New subcommand file or extension to `cmd/fleetctl/fleetctl/hosts.go:17-25`: `fleetctl hosts debug-logging`.
- Client method in `cmd/fleetctl/fleetctl/api.go` calling the new endpoint.

### 6. UI

- Host details page: new action under the existing kebab menu next to "Refetch" — opens a modal with duration picker (15m / 1h / 4h / 24h / 7d).
- Persistent banner on the host page when `orbit_debug_until` is in the future, showing remaining time and a "Disable" action.
- Team settings: the `orbit.debug_logging` key appears automatically in the agent-options JSON editor; no bespoke UI needed for PoC.

### 7. Documentation

- This file.
- `docs/Contributing/reference/audit-logs.md` — two new activity entries.
- REST API reference — new endpoint.
- Changelog entry.

## Open questions

These decisions should be revisited before productionizing. They're acceptable to defer for the PoC.

1. **Sensitive data in debug output.** Before exposing this to admins through the UI, audit orbit's debug-level log statements for enrollment secrets, tokens, or URL query parameters. If any sensitive values are logged, scrub them at the log call site.
2. **Rate-limiting the host endpoint.** A noisy automation that flips debug every minute would churn osqueryd restarts every 30s. Consider a minimum-interval guard (e.g., reject if the previous toggle was < 5 minutes ago).
3. **Startup-flag precedence.** Should `ORBIT_DEBUG=1` at startup act as a floor (server can only raise, never lower)? Current design lets server wins — simpler, but surprises operators who set the env var expecting it to stick.
4. **Per-host "force off" override.** We picked one-way (force-on only) for simplicity. If customers ask for a way to exclude a single host from a team-wide debug sweep, revisit.
5. **Category/component filtering.** Orbit has several subsystems (TUF updater, MDM migrator, Fleet Desktop, extensions); a single global level is coarse. A future enhancement could scope debug to a subsystem.
