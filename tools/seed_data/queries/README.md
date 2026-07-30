> [!NOTE]
> **Prefer [`dibble`](../../dibble/README.md) for seeding reports (formerly "queries").** The equivalent is:
> ```bash
> ./tools/dibble/dibble reports --count N
> ```
> This script is kept for backwards compatibility. It writes directly to
> MySQL (1M rows by default) and is useful for stress-testing the reports
> table specifically; dibble uses the API path.

# Bulk query/report seeder

Direct-MySQL loader that inserts ~1M rows into the `queries` table. Used for
stress-testing scenarios where the API-driven path would be too slow.

## Usage

Assumes the local dev `docker-compose` MySQL (`fleet:insecure@localhost:3306/fleet`).

```bash
go run ./tools/seed_data/queries/seed_queries.go
```
