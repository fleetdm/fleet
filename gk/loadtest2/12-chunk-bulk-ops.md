# Chunk bulk operations and IN lists fleet-wide

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** S–M · **Wave:** 2

## Problem

Queries whose size scales with input size — `IN (...)` lists built from host IDs, multi-row INSERT/UPDATE batches sized by caller input — work in dev and fail at fleet scale: MySQL placeholder limits (65,535), packet size limits, optimizer degradation on huge IN lists, and long row-lock ranges on giant batch writes. We've hit placeholder-limit bugs in production code before; the class keeps reappearing because nothing prevents it.

## Impact

Hard failures (errors on bulk operations targeting many hosts — label application, fleet transfers, batch deletes) and soft failures (lock-heavy mega-statements stalling the hot path) that only appear above customer-scale thresholds, so they ship undetected.

## Proposed fix

- **Sweep:** find dynamic `IN`-list construction and unbounded batch writes in `server/datastore/mysql` (the existing `batchProcessDB`-style helpers show the intended pattern); list offenders in the issue.
- **Helper:** one canonical chunked-execution helper (fixed chunk size, per-chunk transaction or single transaction as caller chooses) so the fix is mechanical.
- **Guardrail:** a lint or datastore-layer assertion (e.g. cap on `sqlx.In` expansion size in dev/test builds) so new unbounded call sites fail in CI instead of at a customer.
- Retrofit the offenders found in the sweep.

## Condition of satisfaction

- Sweep results recorded; all fleet-scale-reachable offenders chunked.
- Guardrail merged — an intentionally unbounded test call site fails CI.
- Bulk host operations (transfer, label apply, delete) succeed against the 400k-host seeded snapshot (issue 16).
