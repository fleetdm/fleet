# charts-backfill

Generates synthetic chart data for development and testing. Writes rows to
`host_hourly_data_blobs` using `ON DUPLICATE KEY UPDATE`, so it is safe to
re-run.

## Usage

```bash
go run ./tools/charts-backfill --dataset uptime --days 30
go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
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
| `--host-ids` | all hosts | Comma-separated host IDs to include |
| `--mysql-dsn` | local dev | MySQL connection string |

## Datasets

- **Hourly blob** (default): 24 rows/day per entity, one per hour.
- **Daily blob** (`cve`): one row/day with `hour = -1` (whole-day sentinel).

Density (fraction of hosts marked active) varies by dataset — see
`densityRange` in `main.go`.
