# Load test metrics

Collect and compare AWS CloudWatch metrics for Fleet load test environments. Use these
scripts to capture a point-in-time synopsis of a running load test and to diff runs
against each other to catch regressions release-over-release.

- `collect-metrics.sh` — discovers a load test environment's AWS resources from its
  Terraform workspace name, pulls CloudWatch metrics averaged over a lookback interval,
  and writes a `.json` data file plus a human-readable `.md` synopsis (with threshold alerts).
- `compare-metrics.sh` — diffs two or more runs side by side and flags deltas as
  `ok` / `WARN` / `ALERT`.

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
```

Key flags (`--help` for the full list):

| Flag | Meaning |
|------|---------|
| `-w, --workspace` | Terraform workspace name (required). AWS resource names are derived from it. |
| `-i, --interval`  | Lookback window: `<N>h`, `<N>m`, or a bare integer (hours). Default `3h`. |
| `-c, --category`  | File the run under a category: `baseline` \| `migration` \| `mdm`. |
| `-o, --output`    | Override the output file path. |
| `-r, --region`    | AWS region. Default `us-east-2`. |

Output lands in `runs/[<category>/]<workspace>/<workspace>-<timestamp>-<interval>.json`
alongside a matching `.md` synopsis.

> The `--workspace` value is not free-form — `collect-metrics.sh` derives AWS resource
> names from it (`fleet-<ws>-backend`, `fleetdm-<ws>-mysql`, `fleet-<ws>-redis`, …), so it
> must match the actual Terraform workspace.

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

## Run organization

Historical runs live under `runs/`, grouped by what the load test exercised:

| Category | `runs/` subfolder | Workspace naming convention | Examples |
|----------|-------------------|-----------------------------|----------|
| **Baseline** — per-release branch load test | `runs/baseline/`  | `<version>loadtest`            | `486loadtest` |
| **Migration** — n-1 → n schema migration    | `runs/migration/` | `<n-1>to<n>mig`                | `485to486mig` |
| **MDM** — platform-specific MDM load test   | `runs/mdm/`       | `<version><platform>` / `<platform>-release` | `483applemdm`, `486-windows` |

The category subfolder is purely for human organization; the scripts don't depend on it.
Keeping workspace names to these conventions is what makes `--filter` a reliable category
selector.

## Submitting results

After a load test, commit the run so the history stays useful for future comparisons:

1. Collect with the right category so the files land in the correct folder, e.g.
   `./collect-metrics.sh --workspace 486loadtest --category baseline`.
2. Commit both the `.json` (data) and `.md` (synopsis) for the run under `runs/<category>/<workspace>/`.
   For multi-step tests (e.g. MDM), a short `REPORT.md` summarizing the runs is welcome too.
3. Open a PR against `main` with the new files. Keep it to the run artifacts — no script changes.