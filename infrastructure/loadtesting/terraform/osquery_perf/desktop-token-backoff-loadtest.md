# Load test: Fleet Desktop exponential backoff (#45624 / #44816)

Procedure for load testing the Fleet Desktop `checkToken` exponential backoff
fix (story [#45624], original incident [#44816]) using the
**Deploy Loadtest - Osquery Perf** GitHub Action
(`.github/workflows/loadtest-osquery-perf.yml`).

This is an **A/B test**: hold a fixed population of hosts stuck with invalid
device tokens, and flip only the retry behavior — flat hammer (pre-fix) vs
exponential backoff (post-fix) — to measure the difference in server/DB load.

> The fix is client-side, so there is nothing new on the server to test. The
> value here is reproducing the pre-fix storm and showing the fix flattens it.
> `osquery-perf` reuses the production `orbit/pkg/backoff` package, so the
> simulated cadence matches the real client.

## Simulator flags (added on branch `loadtest-desktop-token-fail-45624`)

| Flag | Type | Default | Purpose |
|---|---|---|---|
| `desktop_token_fail_prob` | float `[0,1]` | `0.0` | Probability that an orbit/Fleet-Desktop host presents an invalid device token (the storm population). Inert at `0`. |
| `desktop_token_permanent_fail` | bool | `false` | `true` → failing hosts never self-heal (stable steady-state storm, required for a clean A/B). `false` → they recover at the next hourly token rotation (transient). |
| `desktop_token_backoff` | bool | `true` | The A/B variable. `true` → exponential backoff (post-#45624). `false` → flat 5s retry (pre-fix storm). |

Bool flags must use the `=true`/`=false` form inside the `extra_flags` JSON
list (a bare flag works, but `["--flag","true"]` silently breaks parsing of
everything after it).

### Sizing

`desktop_token_fail_prob` is rolled **per host, only on orbit hosts**, so the
failing fraction of the whole fleet ≈ `orbit_prob × desktop_token_fail_prob`.
Set `--orbit_prob=1` so `desktop_token_fail_prob` maps directly to the failing
fraction. Total simulated hosts = `loadtest_containers × --host_count`.

## Prerequisites

1. The infra workspace must already exist (the workflow hard-blocks otherwise);
   deploy it via the infra workspace/workflow first.
2. The branch must be pushed to `origin` so the workflow can check it out via
   `git_tag_branch` (this is a feature branch, **not** merged to `main`):
   `git push origin loadtest-desktop-token-fail-45624`.

## Run A — baseline (pre-fix flat hammer)

Trigger the workflow (`workflow_dispatch`) with:

| Input | Value |
|---|---|
| `terraform_workspace` | your workspace |
| `git_tag_branch` | `loadtest-desktop-token-fail-45624` |
| `loadtest_containers` | e.g. `10` |
| `terraform_action` | `apply` |
| `extra_flags` | see below |

```json
["--orbit_prob","1","--host_count","2000","--start_period","10m","--desktop_token_fail_prob","0.3","--desktop_token_permanent_fail=true","--desktop_token_backoff=false"]
```

→ 20,000 hosts, ~6,000 permanently failing, retrying flat every 5s.

Let it reach steady state, then capture metrics (below).

## Run B — fix (exponential backoff)

Same inputs, identical `extra_flags` **except** `--desktop_token_backoff=true`:

```json
["--orbit_prob","1","--host_count","2000","--start_period","10m","--desktop_token_fail_prob","0.3","--desktop_token_permanent_fail=true","--desktop_token_backoff=true"]
```

Run on the same workspace/scale as Run A so the only variable is the backoff
behavior.

## What to measure

Compare A vs B on the loadtest dashboards:

- Request rate (QPS) to the device endpoint `GET /api/latest/fleet/device/:token/desktop`
- MySQL CPU, active connections, and QPS
- Fleet server CPU / request latency

**Expected result:** identical host count, but Run B's device-endpoint QPS and
DB load collapse versus Run A. A permanently-failing host settles to ~1 request
per 5 min (the backoff cap) vs ~1 per 5s flat — a ~60× per-host reduction,
multiplied across the failing population.

## Optional Run C — transient realism

Drop `--desktop_token_permanent_fail` (defaults to `false`). Failing hosts then
self-heal at their next hourly rotation, so the storm is a decaying recovery
wave rather than a flat plateau. Useful to confirm the wave stays bounded;
sample the *curve*, not a steady-state number. Not part of the head-to-head.

## Teardown

Re-run the workflow with `terraform_action=destroy` on the same
`terraform_workspace` when finished — the load test runs real AWS
infrastructure.

## Notes

- Defaults keep the feature inert (`desktop_token_fail_prob=0.0`), so existing
  load tests are unaffected.
- This branch is for load testing only and is not intended to merge to `main`.

[#45624]: https://github.com/fleetdm/fleet/issues/45624
[#44816]: https://github.com/fleetdm/fleet/issues/44816
