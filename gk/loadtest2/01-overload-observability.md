# Overload observability: pool stats, per-endpoint metrics, shed counters

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** S · **Wave:** 1

## Problem

We cannot see overload coming. There are no exported gauges for DB connection pool pressure (`sql.DBStats`: `InUse`, `WaitCount`, `WaitDuration`), no per-endpoint request rate/latency breakdown oriented around the agent hot path, and no counter infrastructure for load shedding (which doesn't exist yet). Every threshold we pick for the shedding work (issues 02/04/05) would be a guess, and the first signal of bad tuning would be a customer reporting stale hosts.

## Impact

Blocks all admission-control work. Also blocks the local cost-budget harness (issue 18), which reads these same metrics. Without it, overload incidents at a large customer are diagnosed blind.

## Proposed fix

- Export `sql.DBStats` fields as Prometheus gauges for both primary and replica pools (`server/datastore/mysql`), sampled periodically.
- Per-endpoint (route-pattern, not raw path) request count, in-flight count, and latency histograms, with the osquery/MDM agent endpoints individually identifiable.
- Reserve and register `shed_requests_total{endpoint_tier, reason}` now so waves 2–3 increment it rather than inventing names.
- Document a "server overload" dashboard recipe (even just a starter Grafana JSON in `tools/`) and the three signals an operator should page on: pool wait duration, in-flight saturation, shed rate.
- Confirm slow-query logging guidance exists in the deployment docs; add if missing.

## Condition of satisfaction

- Metrics visible in a local `make serve` + Prometheus scrape with documented names.
- Dashboard starter checked in; issue 18's harness can read per-check-in DB time from these metrics alone.

## Evidence

Existing partial coverage: otel middleware in `server/service/middleware/otel`. This issue fills the pool/shed/hot-path gaps, not a metrics rewrite.
