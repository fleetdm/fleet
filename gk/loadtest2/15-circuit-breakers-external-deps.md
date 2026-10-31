# Circuit breakers and bounded concurrency for external dependencies

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Wave:** 3

## Problem

Fleet calls external services — APNs, Windows push, VPP/ABM, NVD and other vuln feeds, SSO IdPs, Jira/Zendesk — from both request paths and jobs. When one is slow (not down — slow is worse), callers block, goroutines and connections accumulate, and an external dependency's bad day becomes Fleet's outage. At 400k hosts the multiplier is severe: an APNs slowdown during a profile rollout means hundreds of thousands of queued pushes holding resources.

## Impact

Cross-cutting availability risk that no amount of DB/shedding work covers, because the stall happens outside the DB. Jira/Zendesk already got defensive 429 handling (`server/service/externalsvc/`); the pattern isn't uniform.

## Proposed fix

- **Inventory** external call sites; for each, verify/retrofit: aggressive timeouts (no default-client infinite waits), bounded concurrency (a worker pool or semaphore per dependency — fan-out work queues rather than spawning unbounded goroutines), and a circuit breaker (fail fast after sustained failures; half-open probes to recover).
- One shared breaker implementation with per-dependency metrics: `external_dep_state{dep}`, failure counts, trip events — wired to the issue 01 dashboard.
- **Degradation semantics per dependency** (the design decision that matters): APNs down → queue pushes, deliver on recovery, UI shows delivery pending; NVD down → serve existing vuln data, skip the sync, alert; IdP down → clear login error, API-token auth unaffected.
- Breakered work that was shed must land in a bounded queue or be deferred to the next job run — not dropped silently and not accumulated unboundedly (ties into issues 08 and 11).

## Condition of satisfaction

- Inventory table with timeout/concurrency-bound/breaker status per dependency; gaps retrofitted.
- Chaos test on the local rig: black-hole APNs (drop packets, don't refuse connections) during a simulated profile rollout — server stays healthy, goroutine count stays bounded, pushes deliver after connectivity returns.
