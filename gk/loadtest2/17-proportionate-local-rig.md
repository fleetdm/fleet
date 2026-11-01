# Proportionate local load test rig

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Track:** local scale testing · **Depends on:** 16

## Problem

Validating 400k-host behavior currently requires the AWS terraform load test environment — too slow and expensive for per-PR or per-feature use, so it runs rarely and scale regressions ship between runs. Naive local testing (a few hundred hosts against an unconstrained server) proves nothing: the server is never under proportionate pressure, so saturation behaviors (pool waits, queueing, lock contention) are unreachable.

## Impact

Enables cheap local validation for issues 02, 04, 05, 07, 13, 14, 15 — each of their conditions of satisfaction references this rig. Without it, every one of those needs an AWS run to verify.

## Proposed fix

A docker-compose overlay in `tools/loadtest/` applying three scaling techniques together:

1. **Shrink the server to a known fraction of production** — CPU limits on the fleet containers, small `innodb_buffer_pool_size`, reduced `max_open_conns` — so saturation appears at laptop-achievable load. Always run **2 fleet containers**: several bug classes (schedule locking, cache coherence, shedding coordination-free behavior) only exist with multiple servers.
2. **Compress time instead of adding hosts** — request rate is hosts ÷ interval, so drive osquery-perf with shortened `-query_interval`, `-config_interval`, `-logger_tls_period`, `-mdm_check_in_interval`; ~2k active hosts at 6s distributed interval generates the check-in QPS of ~200k at 10min.
3. **Run against the seeded snapshot (issue 16)** so tables, indexes, and buffer-pool behavior are at full scale even though active hosts aren't — this closes the artificially-warm-cache gap that time compression alone leaves.

Ship named scenarios as make targets or scripts:
- `steady-state` — baseline for the issue 18 harness.
- `overload` — ramp to 3–5× capacity (validates issues 02/04/05: shedding, bounded p99, recovery).
- `thundering-herd` — `-start_period 0` mass simultaneous action + full-fleet profile push (validates issue 07).
- `dependency-brownout` — black-holed APNs during rollout (validates issue 15).

Document what the rig can and cannot conclude: relative regressions, saturation behavior, and scaling slopes transfer to production; **absolute capacity numbers do not** — those remain the terraform environment's job (release-cadence baselines in `tools/loadtest/metrics/runs/` continue).

## Condition of satisfaction

- `make loadtest-local SCENARIO=overload` (or equivalent) runs end-to-end on a MacBook against the 100k snapshot in under 30 minutes.
- Rig reproduces a known past scale incident class (e.g. a reverted hot-path regression) that plain local testing misses — proof it has teeth.
- README documents scenarios, knobs, and the interpretation limits above.
