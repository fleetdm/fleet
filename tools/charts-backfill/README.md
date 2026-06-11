# charts-backfill

Generates synthetic chart data for development and testing. Writes rows to
`host_scd_data` using `ON DUPLICATE KEY UPDATE`, so it is safe to re-run.
All writes use the roaring bitmap encoding (`encoding_type = 1`).

## Usage

```bash
go run ./tools/charts-backfill --dataset uptime --days 30
go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
go run ./tools/charts-backfill --dataset cve --days 30 --use-tracked-cves
go run ./tools/charts-backfill --dataset cve --days 30 --entity-ids CVE-2024-1,CVE-2024-2
go run ./tools/charts-backfill --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dataset` | `uptime` | Dataset name (`uptime`, `policy`, `cve`, ...) |
| `--days` | `30` | Number of days to backfill |
| `--start-date` | `now - days` | Start date (`YYYY-MM-DD`) |
| `--entity-ids` | `""` | Comma-separated entity IDs (e.g. CVE IDs); `""` for non-entity datasets |
| `--use-tracked-cves` | `false` | For `--dataset cve`, auto-discover entity IDs from the production tracked-CVE query (joins `software_cve` / `operating_system_vulnerabilities` against the curated software matchers; requires vulnerability data to be populated). Overrides `--entity-ids`. |
| `--host-ids` | all hosts | Comma-separated host IDs to include |
| `--mysql-dsn` | local dev | MySQL connection string |

## Datasets

Backfill mode matches the live collector's sample strategy for each dataset:

- **Accumulate, hourly** (default; `uptime`, `policy`): 24 independent rows
  per day per entity, each a fresh random sample. `valid_to` is set to one
  hour past `valid_from`.

- **Snapshot, state-segment** (`cve`): per-entity state-segment rows shaped
  like real CVE data. Each entity gets an initial host set; for each
  subsequent day, with ~5% probability the set is *churned* (~10% drop, ~10%
  add). Each contiguous run of unchanged days collapses to a single row.
  The final segment per entity leaves `valid_to` at the open sentinel so the
  live collector compares against it on its next tick instead of inserting
  over the top. Pair with `--use-tracked-cves` to mirror production CVE
  selection.

Density (fraction of hosts marked active) for the initial sample varies by
dataset — see `densityRange` in `main.go`. Snapshot churn parameters
(`snapshotFlipsPerDayPerEntity`, `snapshotChurnFraction`) are also defined
there.
