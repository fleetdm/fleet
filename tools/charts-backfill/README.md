# charts-backfill

Generates synthetic chart data for development and testing. Writes rows to
`host_scd_data` using `ON DUPLICATE KEY UPDATE`, so it is safe to re-run.

## Usage

```bash
# Realistic CVE profile, default catalog size (1000 tracked CVEs):
go run ./tools/charts-backfill --dataset cve --days 30

# Realistic CVE profile, larger catalog for a more demanding test:
go run ./tools/charts-backfill --dataset cve --days 30 --tracked_cve_count 2000

# Legacy worst-case profile (uniform daily churn, independent host samples):
go run ./tools/charts-backfill --dataset cve --days 30 --profile worst-case

# Non-CVE datasets (the --profile flag is ignored):
go run ./tools/charts-backfill --dataset uptime --days 30
go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
go run ./tools/charts-backfill --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dataset` | `uptime` | Dataset name (`uptime`, `policy`, `cve`, ...) |
| `--days` | `30` | Number of days to backfill |
| `--start-date` | `now - days` | Start date (`YYYY-MM-DD`) |
| `--entity-ids` | `""` | Comma-separated entity IDs (e.g. CVE IDs). Used by the `worst-case` profile; ignored by `realistic`. |
| `--host-ids` | all hosts | Comma-separated host IDs to include |
| `--mysql-dsn` | local dev | MySQL connection string |
| `--profile` | `realistic` | Data profile for the cve dataset: `realistic` or `worst-case` |
| `--tracked_cve_count` | `1000` | Catalog size for the `realistic` cve profile |

## CVE profiles

The `cve` dataset supports two profiles:

### `realistic` (default)

Models the shape observed on real customer fleets and Fleet's own load-test
environment (832 tracked CVEs in current state). Generates a catalog of
`--tracked_cve_count` synthetic CVE IDs, then assigns each one:

- A **churn profile**, weighted:
  - `stable` (20%) — one row covering the whole window
  - `single_flip` (40%) — seed + one membership flip at a random day
  - `active` (40%) — seed + 2-13 flips at random days, small per-flip deltas
- A **host-count band**, right-skewed:
  - `narrow` (70%) — affects 0.1-5% of fleet
  - `medium` (25%) — affects 5-25% of fleet
  - `broad` (5%) — affects 25-100% of fleet

The generator then **injects spike days** at the observed rate (~2-3 per
7-day week of the window), promoting 25-35% of still-stable CVEs to a flip
on that day. This models browser/kernel emergency-patch events where many
CVEs in the same software resolve simultaneously.

Each membership flip is a small delta (±1-3 hosts) from the previous state,
not an independent re-sample. Boundary rules: empty host sets force "add",
full host sets force "remove" — preventing narrow-band CVEs from drifting
to empty over many flips.

### `worst-case`

Preserves the original behavior: one row per CVE per day, with a freshly
sampled host set at every day. Useful as a stress-test regression — the
chart cron and its query path see ~30× more rows than the `realistic`
profile produces, exercising worst-case bitmap and write throughput.

## What this generator does NOT model

- **NVD-driven spike events in osquery-perf's live run.** The realistic
  backfill produces static spike history; the live cron after backfill
  ends operates on whatever `software_cve` looks like. Trigger Fleet's NVD
  sync mid-test to get live spike behavior.
- **Cron-batch quantization.** Real backgrounds have recurring hourly
  counts (e.g., `30 / 185 / 193 / 215` per hour) driven by Fleet's cron
  timing; the simulated background uses smooth small-delta evolution.
  Row volume is similar; timing structure differs.
- **Patch-Tuesday clustering.** Real software updates cluster temporally.
- **Host onboarding/offboarding.** Real fleets gain and lose hosts
  continuously; this generator treats the host roster as static.

## Host-count scaling

Under tracked-CVE scoping (#45247), the chart cron's catalog is bounded
to a few hundred to a few thousand CVEs. The dominant chart-cron stressor
is host count, not CVE count. Per-pass bitmap memory:

```
bitmap_bytes_per_cve = ceil(host_count / 8)
cron_pass_bytes      = bitmap_bytes_per_cve × tracked_cve_count

  10k hosts × 1k CVEs   ≈ 1.25 MB
 100k hosts × 1k CVEs   ≈ 12.5 MB
 100k hosts × 2k CVEs   ≈ 25 MB
```

To push the chart cron, scale `--hosts` on osquery-perf, not
`--tracked_cve_count` here. The CVE-count knob caps out at a few thousand
under tracked-CVE scoping; the host-count knob has no such cap.

## Datasets

- **Hourly blob** (default): 24 rows/day per entity, one per hour.
- **Daily blob** (`cve`): variable rows/day, depending on profile.

Density (fraction of hosts marked active) for the worst-case CVE profile
and the other datasets is set in `densityRange` in `main.go`.
