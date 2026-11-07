# Per-check-in cost budgets and branch-vs-main regression harness

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Track:** local scale testing · **Depends on:** 01, 16, 17

## Problem

We have no scale-invariant definition of "this change made the hot path more expensive." Absolute capacity doesn't extrapolate from small tests — but per-host marginal cost does: a PR that doubles DB time per check-in at 1k hosts doubles it at 400k. Without measured budgets, hot-path regressions are only caught by the occasional AWS run or by a customer.

## Impact

This is the enforcement mechanism that makes "develop for 400k on a laptop" real. It converts the whole load-protection effort from a one-time cleanup into a ratchet that can't silently slip.

## Proposed fix

**Budgets (scale-invariant, per simulated host):**
- writes per idle check-in (target: ~0 — issue 14 establishes the baseline)
- DB time per check-in (ms), reads per check-in
- lock wait per check-in (issue 13 wires the metric)
- bytes stored per host per day

**Harness** (script in `tools/loadtest/`):
1. Restore seeded snapshot (16), start the constrained rig (17), run the `steady-state` scenario against **main**, collect budget metrics (from issue 01's instrumentation + MySQL counters).
2. Repeat against the branch. Report per-budget deltas; fail over threshold (e.g. >10% DB time, any new idle-check-in write).
3. **Slope mode:** run the same workload at two seeded scales (20k and 100k snapshots) and compare per-request cost. Flat ⇒ no superlinear term in this range; upward slope ⇒ one exists (missing index, growing scan) and won't survive 400k. Extrapolate the exponent, never the capacity — and treat a flat slope as strong evidence, not proof: effects that only begin past 100k (plan flips, buffer-pool cliffs) remain the AWS runs' job to catch, which is why the 400k snapshot exists as a periodic deep check.

**Adoption path:** manual/on-demand first (`make loadtest-compare`), required for PRs touching check-in-path packages once stable (nightly job or opt-in CI label), with results posted as a PR comment table.

## Condition of satisfaction

- `make loadtest-compare` produces a budget-delta table for the current branch in under 30 minutes on a laptop.
- Demonstrated catch: introduce a deliberate unconditional write into the check-in path on a test branch — the harness flags it.
- Slope mode demonstrated on one known-superlinear historical query shape.
