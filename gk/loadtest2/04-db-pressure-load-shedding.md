# Adaptive load shedding driven by DB pressure

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Wave:** 2 · **Depends on:** 01, 02, 03

## Problem

The concurrency cap (issue 02) bounds work per container, but the shared bottleneck is the database: every container can be under its cap while MySQL is drowning (pool saturated, lock waits, replica lag). We need shedding driven by the actual bottleneck signal, not just local concurrency.

## Impact

Without it, DB saturation shows up as uniform latency collapse across every endpoint and container at once. With it, the server sheds deferrable agent work early and keeps functioning.

## Proposed fix

Each container samples its own DB pressure locally — no coordination plane. Because the DB is shared, local measurements reflect cluster-wide pressure; if every container is hot, every container sheds, which is the desired global behavior with zero coordination machinery.

Signals (from issue 01's instrumentation):
- Pool saturation: `InUse ≈ MaxOpenConns` with `WaitCount`/`WaitDuration` climbing.
- Latency: EWMA of datastore call duration exceeding a healthy baseline (catches "connections free but MySQL slow" — lock contention, replica lag).

Behavior:
- Shed **proportionally**: reject a fraction of sheddable requests scaled to how far past threshold the signal is — not a binary switch.
- **Hysteresis**: require the signal healthy for a sustained window before fully reopening, so it settles instead of oscillating.
- Response is `503` + jittered `Retry-After`; counts into `shed_requests_total{reason="db_pressure"}`.
- Sheddable set starts as the expensive agent endpoints (detail/inventory refresh, log submission — osquery buffers and retries); full tiering arrives with issue 05.
- Explicitly CPU-agnostic: no cluster CPU aggregation. A coordination plane adds lag and a new failure mode exactly when things are melting.

## Condition of satisfaction

- On the proportionate local rig (issue 17): drive the DB to saturation and observe shed rate rising proportionally, admitted-request p99 staying bounded, no oscillation (shed rate settles), and full recovery without restart when load drops.
- Thresholds configurable with sane defaults; behavior documented for operators.

## Evidence

`sql.DBStats` already exposed by the pools in `server/datastore/mysql/mysql.go`; nothing consumes it today.
