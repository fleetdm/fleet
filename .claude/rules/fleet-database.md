---
paths:
  - "server/datastore/**/*.go"
---

# Fleet Database Conventions

## Migration Files
- Location: `server/datastore/mysql/migrations/tables/`
- Naming: `YYYYMMDDHHMMSS_CamelCaseName.go` (timestamp + descriptive CamelCase)
- Every migration that modifies data MUST have a corresponding `_test.go` file. That means changing existing data, adding new data, or
  populating a table
- Simple migrations that only add a table, column, or index do not need one
- Structure:
  ```go
  func init() {
      MigrationClient.AddMigration(Up_YYYYMMDDHHMMSS, Down_YYYYMMDDHHMMSS)
  }
  func Up_YYYYMMDDHHMMSS(tx *sql.Tx) error { ... }
  func Down_YYYYMMDDHHMMSS(tx *sql.Tx) error { return nil }  // always no-op
  ```
- Test pattern: `applyUpToPrev(t)` → set up data → `applyNext(t, db)` → verify
- Create with: `make migration name=YourChangeName`

## Migrations
- Guard schema changes so a failed migration can be retried
- `CREATE TABLE`/`DROP TABLE`: use `IF NOT EXISTS`/`IF EXISTS`
- `ALTER TABLE`: use `columnExists`/`indexExistsTx`/etc from `migration.go`

## Migrations on large tables
Some tables hold hundreds of millions of rows in large deployments. A migration that scans or rewrites one of them can run for hours and block a customer's upgrade (this has happened in production). Any migration that reads or writes one of these tables needs extra scrutiny:

- **One row per host**: `hosts`, `host_mdm`, `host_operating_system`, `host_disks`, `host_display_names`, `host_emails`, `host_issues`
- **Many rows per host** (largest tables): `host_software`, `host_software_installed_paths`, `host_users`, `host_certificates`, `host_script_results`, `label_membership`, `policy_membership`, `scheduled_query_stats`, `host_mdm_apple_profiles`, `host_mdm_windows_profiles`, `host_mdm_managed_certificates`
- **Software inventory**: `software`, `software_titles`, `software_cpe`, `software_cve`, `software_host_counts`, `software_titles_host_counts`
- **Append-only event/queue tables**: `activities`, `upcoming_activities`, `nano_commands`, `nano_command_results`, `nano_enrollment_queue`, `windows_mdm_commands`, `windows_mdm_command_queue`, `windows_mdm_command_results`, `windows_mdm_responses`

This list is not exhaustive — treat any table that scales with host count × per-host entities (software, certificates, profiles, query results) or that accumulates events/commands over time the same way.

For migrations touching these tables:
- Run `EXPLAIN` on every SELECT/UPDATE/DELETE and confirm it uses an index — never rely on the optimizer's default plan. Index statistics are often stale mid-migration, so the optimizer can silently pick a full table scan; consider `FORCE INDEX` on these tables when the right index exists. Do NOT apply `FORCE INDEX` as a blanket rule on small tables — it creates its own bugs.
- Prefer batched updates (loop with LIMIT, log progress) over a single statement that touches every row.
- Avoid DDL that rewrites the whole table (column type changes, etc.) when an online-safe alternative exists.

## Query Building
- Use `goqu` (github.com/doug-martin/goqu/v9) for SQL query building
- Pattern: `dialect.From(goqu.I("table_name")).Select(...).Where(...)`
- NEVER use string concatenation for SQL — parameterized queries only
- The `gosec` linter checks for SQL concatenation (G202)

## Reader vs Writer
- Reads: `ds.reader(ctx)` — may hit a read replica
- Writes: `ds.writer(ctx)` — always hits the primary
- Using the wrong one causes stale reads or replica lag issues

## Testing
- Integration tests require `MYSQL_TEST=1`: `MYSQL_TEST=1 go test ./server/datastore/mysql/...`
- Use `CreateMySQLDS(t)` helper for test datastore setup
- Table-driven tests with `t.Run` subtests

## Transactions
- Inside `withTx`/`withRetryTxx` callbacks, use the transaction argument — NEVER call `ds.reader(ctx)` or `ds.writer(ctx)` inside a transaction (custom linter rule catches this)
- Same applies to any function that receives a `sqlx.ExtContext` or `sqlx.ExecContext` as an argument — use that argument, not the datastore's reader/writer

## Batch Operations
- Use configurable batch size variables for large operations
- Order key allowlists for user-facing sort fields (prevent SQL injection via ORDER BY)
