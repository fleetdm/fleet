# Load test metrics

Collect and compare AWS CloudWatch metrics for Fleet load test environments. Use these
scripts to capture a point-in-time synopsis of a running load test and to diff runs
against each other to catch regressions release-over-release.

- `collect-metrics.sh` — discovers a load test environment's AWS resources from its
  Terraform workspace name, pulls CloudWatch metrics averaged over a lookback interval,
  and writes a `.json` data file plus a human-readable `.md` synopsis (with threshold alerts).
  Covers ECS (Fleet server + loadtest containers), RDS writer/readers (including
  Performance Insights top SQL and active sessions), Redis, ALB (avg/p95/p99 latency,
  target and ELB 5xx, request/traffic volume), Fleet server log errors, and container
  restart health. Each CloudWatch metric records a `data_coverage` ratio (fraction of
  the window with datapoints) so partial-window runs are detectable.
- `compare-metrics.sh` — diffs two or more runs side by side and flags deltas as
  `ok` / `WARN` / `ALERT`. Count metrics (requests, 5xx, evictions, log errors) are
  normalized to per-hour rates using each run's interval, so runs with different
  lookback windows compare fairly. Grading has per-metric materiality floors —
  large percentage swings on near-zero values are not flagged. Values collected
  with <75% datapoint coverage are marked with `*`. Performance Insights top SQL
  is diffed across runs (matched on the full tokenized statement): new statements
  entering the top-N are flagged `NEW`, and load increases ≥50%/≥100% for an
  existing statement are graded WARN/ALERT.

## Requirements

- [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html),
  authenticated against the account hosting the load test environment.
- [`jq`](https://jqlang.github.io/jq/)

## Collecting metrics

```bash
# 3h lookback (default) for the "486loadtest" workspace
./collect-metrics.sh --workspace 486loadtest

# 1h lookback, filed under the mdm category
./collect-metrics.sh --workspace 483applemdm --interval 1h --category mdm

# An arbitrary past window: the 90 minutes ending 2 hours ago
./collect-metrics.sh --workspace 486loadtest --interval 90m --end 2h

# Or pin the end to an exact UTC timestamp
./collect-metrics.sh --workspace 486loadtest --interval 15m --end 2026-08-28T02:28:58Z
# annotate what the run was exercising
./collect-metrics.sh --workspace 483applemdm --category mdm \
  --note "300k hosts, APNs push storm at t+40m"
```

Key flags (`--help` for the full list):

| Flag | Meaning |
|------|---------|
| `-w, --workspace` | Terraform workspace name (required). AWS resource names are derived from it. |
| `-i, --interval`  | Window length: `<N>h`, `<N>m`, or a bare integer (hours). Default `3h`. |
| `-e, --end`       | End of the window: an absolute UTC timestamp (`2026-08-28T02:28:58Z`) or a relative age (`2h`, `90m`). Default: now. Combine with `--interval` to collect any past range. |
| `-c, --category`  | File the run under a category: `baseline` \| `migration` \| `mdm`. |
| `-n, --note`      | Free-form note about the run, embedded as `metadata.note`. Shown in the synopsis and dashboard; never compared. |
| `-o, --output`    | Override the output file path. |
| `-r, --region`    | AWS region. Default `us-east-2`. |

Output lands in `runs/[<category>/]<workspace>/<workspace>-<timestamp>-<interval>.json`
alongside a matching `.md` synopsis. The timestamp is the **end of the window**, not the
moment the script ran, so historical collections are self-describing and sort in window
order; the wall-clock run time is recorded separately as `metadata.collected_at`.

> The `--workspace` value is not free-form — `collect-metrics.sh` derives AWS resource
> names from it (`fleet-<ws>-backend`, `fleetdm-<ws>-mysql`, `fleet-<ws>-redis`,
> `fleet-<ws>-apns-mock`, …), so it must match the actual Terraform workspace.

### MDM runs

Two extra sections appear when the deployment enabled Apple MDM
(`var.enable_apple_mdm`); both are absent otherwise, and `compare-metrics.sh` drops
sections with no data.

`apns_mock` covers the [mock APNs server](../../../cmd/apple-apns-mock), which scales
horizontally (`var.apple_apns_mock_instance_count`):

- `cpu_utilization` / `memory_utilization` — Sum(Utilized)/Sum(Reserved) across every
  running task, i.e. the service-wide average.
- `per_task` — the mean and hottest single container. Worth watching alongside the
  average: one saturated task drops every SSE stream it holds while the service-wide
  figure still looks healthy. A large `spread_pct` means the ALB is not distributing
  connections evenly.
- `network` — RX/TX bytes. Held SSE streams make traffic volume a better throughput
  proxy here than the ALB's request count.
- `container_health` — abnormal stops and start spread, scoped to this service. An
  OOM-killed task disconnects its devices, which then reconnect elsewhere.

`apns_mock_redis` covers the mock's dedicated ElastiCache. Load tracks the push rate and
instance count rather than the connection count — each task holds one subscribe
connection plus a small command pool:

- `cpu_utilization` — `EngineCPUUtilization`. Redis runs commands on one thread, so this
  saturates well before host CPU does.
- `curr_items` — pending pushes awaiting claim, plus one stats key per instance. A rising
  floor means devices are not connecting to collect them.
- `evictions` — must stay at zero. An eviction is a pending push dropped before its
  device reconnected.
- `string_based_cmds` / `pubsub_based_cmds` — SET/GETDEL/INCR and PUBLISH/SUBSCRIBE
  volume, the push throughput as Redis saw it.

## Comparing runs

```bash
# Compare the 2 most recent runs across all categories
./compare-metrics.sh

# Last 4 baseline releases, one run per workspace
./compare-metrics.sh --filter loadtest --depth 4 --unique

# Two specific files
./compare-metrics.sh runs/baseline/485loadtest/485*.json runs/baseline/486loadtest/486*.json
```

`compare-metrics.sh` searches `runs/` **recursively**, so category subfolders are included
automatically. The `--filter` flag matches on the workspace name, which — thanks to the
naming conventions below — doubles as a category selector (`--filter loadtest`, `--filter mig`).

## Visualizing runs

Open [`dashboard.html`](dashboard.html) in a browser and drop the `runs/` folder onto the
page (or use the folder picker). Everything is parsed and rendered locally — no server, no
network, nothing leaves the browser.

- **Overview** — one sparkline card per metric across the selected runs, colored by the
  latest release-over-release movement (green/amber/red, using the same thresholds,
  per-hour normalization, and noise floors as `compare-metrics.sh`), worst first.
- **Detail** — click any card (or pick from the metric dropdown) for a full line chart
  with the absolute threshold drawn in. Any numeric path in the JSON can also be charted
  via the raw-path input.
- Runs are selectable individually, grouped by category; baselines are selected by
  default since migration/MDM runs exercise different workloads.
- A run collected with `--note` shows that note under its entry in the run list and in
  the detail-chart tooltip. Notes are display-only — never charted or diffed.

The dashboard's metric registry mirrors the definitions in `compare-metrics.sh` — keep
them in sync when adding metrics.

## Run organization

Historical runs live under `runs/`, grouped by what the load test exercised:

| Category | `runs/` subfolder | Workspace naming convention | Examples |
|----------|-------------------|-----------------------------|----------|
| **Baseline** — per-release branch load test | `runs/baseline/`  | `<version>loadtest`            | `486loadtest` |
| **Migration** — n-1 → n schema migration    | `runs/migration/` | `<n-1>to<n>mig`                | `485to486mig` |
| **MDM** — platform-specific MDM load test   | `runs/mdm/`       | `<version><platform>` / `<platform>-release` | `483applemdm`, `486-windows` |
| **Historical** — backfilled from the manual results spreadsheet (4.63–4.85) | `runs/historical/` | as recorded in the sheet | `4630loadtestbl` — see [runs/historical/README.md](runs/historical/README.md) |

The category subfolder is purely for human organization; the scripts don't depend on it.
Keeping workspace names to these conventions is what makes `--filter` a reliable category
selector.

## Known caveats in historical runs

Runs collected before August 2026 (≤ 4.89) have two data quirks from since-fixed
collection bugs:

- `alb.target_response_time.Average` is `0` — an over-aggressive rounding step
  flattened seconds-scale averages. Use p95/p99 (present in newer runs) going forward.
- `container_health` start-spread/uptime fields are `null`/empty, and on macOS the
  `fleet_server_errors` window was shifted by the collector's UTC offset.

## Submitting results

After a load test, commit the run so the history stays useful for future comparisons:

1. Collect with the right category so the files land in the correct folder, e.g.
   `./collect-metrics.sh --workspace 486loadtest --category baseline`.
2. **Review `fleet_server_errors.sample_messages` in the `.json` before committing.**
   These are raw server log lines headed for a public repo. The collector redacts
   token-like strings automatically, but eyeball them anyway — don't commit
   anything that looks like a secret, customer data, or an internal URL.
3. Commit both the `.json` (data) and `.md` (synopsis) for the run under `runs/<category>/<workspace>/`.
   For multi-step tests (e.g. MDM), a short `REPORT.md` summarizing the runs is welcome too.
4. Open a PR against `main` with the new files. Keep it to the run artifacts — no script changes.