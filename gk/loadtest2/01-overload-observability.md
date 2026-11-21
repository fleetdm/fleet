# Overload observability: pool stats, per-endpoint metrics, shed counters

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** S · **Wave:** 1

## Problem

We cannot see overload coming. DB pool stats (`sql.DBStats`) are exported via OTEL (`otelsql.RegisterDBStatsMetrics`), but nothing consumes them operationally — no documented alert thresholds or dashboard. There is no per-endpoint request rate/latency breakdown oriented around the agent hot path, and no counter infrastructure for load shedding (which doesn't exist yet). Every threshold we pick for the shedding work (issues 02/04/05) would be a guess, and the first signal of bad tuning would be a customer reporting stale hosts.

## Impact

Blocks all admission-control work. Also blocks the local cost-budget harness (issue 18), which reads these same metrics. Without it, overload incidents at a large customer are diagnosed blind.

## Proposed fix

All instrumentation goes through OTEL — the team is dropping direct Prometheus support, so no new Prometheus-only metrics.

- Verify the existing `otelsql.RegisterDBStatsMetrics` export covers both primary and replica pools and document the metric names; fill gaps rather than re-instrument.
- Per-endpoint (route-pattern, not raw path) request count, in-flight count, and latency histograms, with the osquery/MDM agent endpoints individually identifiable — extend the existing otel middleware (`server/service/middleware/otel`).
- Reserve and register a shed counter (`fleet.shed_requests` with tier/reason attributes) now so waves 2–3 increment it rather than inventing names.
- In-process consumers: the shedding middleware (issues 02/04) reads pool pressure directly from `sql.DBStats` in-process, not via the metrics pipeline — the OTEL export is for operators, not for control decisions.
- Document a "server overload" dashboard recipe (starter Grafana JSON against the OTEL metrics in `tools/`) and the three signals an operator should page on: pool wait duration, in-flight saturation, shed rate.
- Confirm slow-query logging guidance exists in the deployment docs; add if missing.

## Condition of satisfaction

- Metrics visible via a local OTEL collector setup with documented names.
- Dashboard starter checked in; issue 18's harness can read per-check-in DB time from these metrics alone.

## Evidence

Existing coverage: otel middleware in `server/service/middleware/otel`; pool stats already exported via `otelsql.RegisterDBStatsMetrics` (PR review, getvictor). This issue fills the shed/hot-path/dashboard gaps, not a metrics rewrite.
