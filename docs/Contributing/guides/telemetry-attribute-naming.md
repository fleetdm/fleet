# Attribute naming conventions

Per [ADR-0009](../../docs/Contributing/adr/0009-attribute-naming-conventions.md), Fleet uses consistent
attribute names across logs, traces, and metrics. This document covers the practical rules and the
most common attributes.

## Naming rules

All attribute keys must be:

- **Lowercase**
- **Dot-namespaced** for hierarchy (`host.id`, not `host_id`)
- **snake_case** within components (`host.osquery_version`, not `host.osqueryVersion`)

Do not use camelCase, kebab-case, or bare generic names like `id`, `name`, `err`, or `status`.

## Use OTEL semantic conventions first

When an [OTEL semantic convention](https://opentelemetry.io/docs/specs/semconv/general/attributes/) attribute exists, use it:

| Use this | Instead of |
|----------|-----------|
| `http.request.method` | `method` |
| `url.path` | `uri` |
| `http.response.status_code` | `status_code`, `code` |
| `client.address` | `ip_addr` |
| `client.forwarded_for` | `x_for_ip_addr` |
| `error.type` | (already used) |
| `exception.type` | (already used) |
| `exception.message` | (already used) |
| `exception.stacktrace` | (already used) |
| `db.system` | (already used) |

## Common attributes

### Identifiers

| Attribute | Type | Description |
|-----------|------|-------------|
| `host.id` | uint | Database primary key |
| `host.uuid` | string | Hardware UUID from osquery |
| `host.hardware_serial` | string | Device serial number |
| `host.platform` | string | OS platform (darwin, windows, ubuntu, etc.) |
| `user.id` | uint | User database primary key |
| `user.email` | string | User email address |
| `fleet.id` | uint | Fleet (team) database primary key |
| `report.id` | uint | Report (query) database primary key |
| `report.name` | string | Report (query) name |
| `policy.id` | uint | Policy database primary key |
| `policy.name` | string | Policy name |
| `campaign.id` | uint | Live query campaign ID |

### MDM

| Attribute | Type | Description |
|-----------|------|-------------|
| `mdm.profile.uuid` | string | MDM profile UUID |
| `mdm.command.uuid` | string | MDM command UUID |

### Scheduling and background work

| Attribute | Type | Description |
|-----------|------|-------------|
| `cron.name` | string | Cron schedule name |
| `cron.instance` | string | Server instance running the job |
| `cron.type` | string | Trigger type (triggered, scheduled_tick, trigger_poll) |
| `async.task` | string | Async task name |
| `job.id` | uint | Background job ID |
| `job.name` | string | Background job type name |

### Errors

| Attribute | Type | Description |
|-----------|------|-------------|
| `error.message` | string | Error message (replaces bare `err`) |
| `error.internal` | string | Internal error detail, not user-facing |
| `error.uuid` | string | Error UUID for correlation |

### Request context

| Attribute | Type | Description |
|-----------|------|-------------|
| `duration` | time.Duration | Request or operation duration (replaces `took`) |

### Bytes written

Use `response.bytes_written` instead of `bytes_copied`, `bytes_written`, or `written`.

## Metric names

Metric instrument names (not attributes) use the `fleet.` prefix: `fleet.http.client_errors`,
`fleet.http.server_errors`. Attributes on those metrics are unprefixed.

## Cardinality

Attributes with many distinct values (host IDs, UUIDs, emails, SQL text) must not be used as
metric attributes. They are safe for logs and spans only. Rule of thumb: if it can exceed ~100
distinct values, keep it off metrics.
