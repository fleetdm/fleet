# Check-in path cost audit: write-only-on-change and shared-read caching

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Wave:** 3

## Problem

Two per-check-in cost patterns that each multiply by 400k hosts × check-in frequency:

1. **Writes when nothing changed.** The most expensive pattern at scale is a per-host write on every check-in even for unchanged data. Fleet's cached-host-updates / batched seen-time pattern exists to prevent this, but features added since can bypass it — each one adds an unconditional write to every check-in in the fleet.
2. **Per-host computation of shared answers.** Config, policy sets, queries-to-run are often identical for every host in a team/fleet; computing or fetching them per host per request does fleet-sized work for team-sized data.

## Impact

Directly sets baseline DB load per host — the multiplier on everything. Cutting an unconditional write from the check-in path is worth more than most capacity work, because it scales down load at every fleet size.

## Proposed fix

- **Measure first:** on the local rig with steady-state (nothing changing) simulated hosts, capture per-check-in statement logs. The ideal steady-state check-in is ~0 writes; every observed write is a finding.
- For each finding: route through the batched/cached update pattern, add a changed-check before writing, or justify in writing why the write must be unconditional.
- **Reads:** rank check-in-path queries by frequency; any whose result is host-independent (per-team config, schedules, policy definitions) gets cached (in-memory with TTL/invalidation, or Redis where cross-container consistency matters).
- **Lock in the gain:** issue 18's budget harness asserts "writes per idle check-in" so the number can't silently regress; the scale checklist (issue 19) makes write-on-change the default for new check-in features.

## Condition of satisfaction

- Steady-state write audit published: per-check-in statement list with each write eliminated, batched, or justified.
- "Writes per idle check-in" and "DB reads per check-in" tracked in the regression harness with agreed budgets.
