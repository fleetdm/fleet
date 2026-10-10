# Epic: Server load protection for large fleets

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering`, `epic` · **Milestone:** tracks through v5.0.0

## Problem

Large customers are about to enroll fleets in the 100k–400k host range. Today the server has no overload protection beyond fixed-quota rate limits on a handful of auth endpoints (`server/service/handler.go`): no load shedding, no backpressure signaling, no admission control. Under sustained overload the failure mode is cascading — queued requests, DB pool exhaustion, lock pileups — rather than graceful degradation. Several code patterns (unbounded fan-out, select-all-hosts jobs, unbounded growth tables) are fine at 10k hosts and incident-grade at 400k.

## Impact

A single large customer onboarding, config push, or incident-triggered retry storm can take the server fully down for all tenants of a deployment, including the UI an operator needs to respond. Severity: critical at target scale; today it's latent.

## Goal

No matter how high the load gets, the server keeps functioning: it sheds low-priority work gracefully, protects the UI and critical agent flows, signals backoff that agents honor, and recovers without operator intervention. v5.0.0 (≈6 weeks out) is the window to make the breaking API changes that make this uniform across the product.

## Child issues

**Wave 1 — see it, bound it (start now)**
- 01 Overload observability
- 02 In-flight concurrency cap middleware
- 03 osquery error paths must never return node_invalid
- 19 Scale checklist + load-test gate in templates

**Wave 2 — shed and smooth**
- 04 DB-pressure adaptive load shedding
- 05 v5.0: uniform overload semantics + endpoint priority tiers (breaking)
- 06 fleetd honors Retry-After
- 07 Fan-out safety: splay + reconcile convergence
- 08 Background job chunking audit
- 12 Chunk bulk ops & IN lists

**Wave 3 — harden the base (v5.0-aligned breaking changes land here)**
- 09 v5.0: mandatory pagination on all list endpoints (breaking)
- 10 v5.0: payload size caps on agent-submitted data (breaking)
- 11 Retention for activity-growth tables
- 13 Hot-table transaction/lock discipline
- 14 Check-in path cost audit (write-on-change + shared-read caching)
- 15 Circuit breakers on external dependencies

**Local scale testing (parallel track, enables everything above)**
- 16 Seeded-scale database snapshot tooling
- 17 Proportionate local load test rig
- 18 Per-check-in cost budgets + branch-vs-main regression harness

## Evidence

- Existing full-scale infra: `infrastructure/loadtesting/terraform`, baselines in `tools/loadtest/metrics/runs/` (4.83, 4.89, 4.90)
- Prioritization matrix: https://claude.ai/code/artifact/f6756b05-8467-4571-8131-331483591e43
