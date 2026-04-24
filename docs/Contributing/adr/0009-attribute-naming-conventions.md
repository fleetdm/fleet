# ADR-0009: Attribute naming conventions for logs, traces, and metrics

## Status

Accepted

## Date

2026-03-27

## Context

Fleet uses structured logging, distributed tracing, and metrics for observability ([ADR-0005](0005-opentelemetry-standardization.md)). These signals share a common need: well-named attributes that engineers can use to search, filter, and correlate data.

Consistent attribute naming is valuable regardless of the backend. Whether logs go to AWS CloudWatch, traces go to an OTEL collector, or both end up in the same system, using the same key for the same concept (e.g., `host.id` in both a log line and a span) lets engineers correlate across signals.

Fleet supports self-hosted (on-prem) deployments where customers pipe logs, traces, and metrics into their own observability systems. This makes attribute names a customer-facing surface. While only infrastructure engineers at customer organizations interact with this data directly, inconsistent or poorly named attributes make Fleet harder to operate and monitor.

The codebase currently uses inconsistent attribute names:

- **Mixed naming styles**: `team_id` (snake_case), `numHosts` (camelCase), `ingestion-err` (kebab-case), `host.id` (dot-notation)
- **Same concept, different names**: `bytes_copied`, `bytes_written`, and `written` all represent bytes written to a response
- **Same entity, different key formats**: `host_id` vs `host-id` for the same numeric identifier
- **Overly generic keys**: `err`, `name` are ambiguous and will cause confusion as telemetry volume grows

Industry best practices make a strong case for standardized attribute naming:

1. **Cross-signal correlation**: When logs, traces, and metrics share the same attribute keys (e.g., `http.request.method` instead of `method`), observability backends can automatically correlate across signals. Even when signals live in separate systems, shared keys make manual correlation possible by searching for the same value across tools.

2. **Tooling interoperability**: Observability backends (SigNoz, Grafana, etc.) build dashboards and queries around well-known attribute names. Using `http.response.status_code` gets out-of-the-box visualizations; using `status` or `code` does not.

3. **Specification guidance**: The OpenTelemetry semantic conventions define a well-tested vocabulary for common concepts. Adopting these names where they fit gives us a ready-made standard rather than inventing our own.

## Decision

Fleet will adopt a tiered attribute naming convention for all telemetry signals (logs, traces, metrics):

### Naming format

All attribute names must be:
- Lowercase
- Dot-namespaced for hierarchy (e.g., `host.id`, not `host_id`)
- Snake_case within namespace components (e.g., `host.osquery_version`)
- No camelCase, kebab-case, or SCREAMING_CASE

### Tier 1: Use existing semantic conventions where they fit

When an OpenTelemetry semantic convention attribute exists and matches the semantics, use it directly. Key examples:

| Attribute | Replaces |
|-----------|----------|
| `http.request.method` | `method` |
| `url.path` | `uri` |
| `error.type` | (already used) |
| `exception.type` | (already used) |
| `client.address` | `ip_addr` |

### Tier 2: Domain-first custom attributes (no company prefix)

For Fleet-specific concepts, use the same dot-namespaced style without a company prefix. This follows the "domain-first, never company-first" [guidance](https://opentelemetry.io/blog/2025/how-to-name-your-span-attributes/) and keeps attributes concise:

| Namespace | Examples |
|-----------|----------|
| `host.*` | `host.id`, `host.uuid`, `host.platform` |
| `user.*` | `user.id`, `user.email` |
| `team.*` | `team.id` |
| `query.*` | `query.id`, `query.name`, `query.sql` |
| `cron.*` | `cron.name`, `cron.instance` |
| `job.*` | `job.id`, `job.name` |

Some of these (e.g., `host.id`, `host.name`) are already defined as semantic convention resource attributes. Using the same names is intentional.

### Metric names: `fleet.` prefix

Metric instrument names (not attributes on those metrics) use the `fleet.` prefix since they are registered globally and must be distinguishable from other libraries (e.g., `fleet.http.client_errors`).

### Constants vs inline strings

Attribute keys that appear in more than one place or that are used for dashboards and alerts should be defined as typed constants. Each domain owns its constants. For example:

```go
const (
    HostID   = attribute.Key("host.id")
    HostUUID = attribute.Key("host.uuid")
)
```

One-off attributes that are local to a single function (e.g., `"bytes_remaining"` in a download retry loop) can use inline strings. The naming convention still applies to inline strings.

### Cardinality

Attributes with more than ~100 distinct values (e.g., `host.id`, `user.id`, `query.sql`) must not be used as metric attributes. They are safe for spans and logs only.

## Consequences

### Positive

- **Cross-signal correlation**: Same attribute keys across logs, traces, and metrics enable correlation whether signals share a backend or not
- **Tooling interoperability**: Semantic convention attributes get out-of-the-box dashboards and queries in standard tooling
- **Compile-time safety**: Typed constants catch typos and make attribute discovery easy via IDE autocomplete
- **Incremental migration**: Existing code can be updated file-by-file as it is touched

### Negative

- **Collision risk**: Unprefixed names could theoretically collide with future semantic convention additions. In practice this risk is low for domain-specific names like `team.id` or `cron.name`, and a single-attribute migration is straightforward if it occurs
- **Migration effort**: Existing attribute names across the codebase need updating, though this can be done incrementally
- **Dashboard updates**: Observability queries referencing old attribute names will need updating after each batch of changes

## Alternatives considered

### Company prefix on all attributes (`fleet.*`)

- **Pros**: Zero collision risk with semantic conventions, clear provenance
- **Cons**: Verbose (`fleet.host.id` vs `host.id`), higher friction for engineers, goes against "domain-first" guidance
- **Rejected because**: The adoption cost outweighs the low collision risk. Engineers are more likely to follow a convention that is concise.

### No convention (status quo)

- **Pros**: No migration effort
- **Cons**: Growing inconsistency, broken cross-signal correlation, no tooling interop
- **Rejected because**: Inconsistent naming actively degrades the value of the telemetry data we are already collecting.

## References

- [Telemetry attribute naming guide](../guides/telemetry-attribute-naming.md) - Fleet's practical conventions for day-to-day use
- [OTEL semantic conventions: naming](https://opentelemetry.io/docs/specs/semconv/general/naming/)
- [How to name your span attributes (OTEL blog, 2025)](https://opentelemetry.io/blog/2025/how-to-name-your-span-attributes/)
- [OTEL general attributes](https://opentelemetry.io/docs/specs/semconv/general/attributes/)
- [ADR-0005: Standardize on OpenTelemetry for observability](0005-opentelemetry-standardization.md)
