# charts-collect

Fetches live data from a Fleet instance via the REST API and writes chart rows
into a local database. Designed to run hourly via cron.

## What it collects

- **Uptime** — fetches currently online hosts and ORs them into the current
  hour's `host_hourly_data_blobs` row (`dataset='uptime'`).
- **CVE** — fetches per-host vulnerabilities, inverts into per-CVE host
  bitmaps, and reconciles into `host_scd_data` (`dataset='cve'`). Unchanged
  CVEs keep their open row; changed bitmaps close the prior-day row and open
  a new one for today; intra-day changes overwrite today's row via ODKU.

## Usage

```bash
go run ./tools/charts-collect \
  --fleet-url https://dogfood.fleetdm.com \
  --fleet-token <token>

go run ./tools/charts-collect \
  --fleet-url https://dogfood.fleetdm.com \
  --fleet-token <token> \
  --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
```

## Flags and env vars

| Flag | Env | Description |
|------|-----|-------------|
| `--fleet-url` | `FLEET_URL` | Fleet server URL (required) |
| `--fleet-token` | `FLEET_TOKEN` | Fleet API token (required) |
| `--mysql-dsn` | `MYSQL_DSN` | Full MySQL DSN |

If `--mysql-dsn` / `MYSQL_DSN` is not set, the DSN is assembled from the same
env vars used by the fleet server (so the same values can be reused, e.g. via
Render `fromService`):

- `FLEET_MYSQL_ADDRESS`
- `FLEET_MYSQL_USERNAME`
- `FLEET_MYSQL_PASSWORD`
- `FLEET_MYSQL_DATABASE`

## Notes

- SCD encoding constants (`9999-12-31` open sentinel, batch size) mirror
  `server/chart/internal/mysql/scd.go`. Keep in sync when either side changes.
- Errors in one collector (uptime/cve) are logged but do not block the other.
