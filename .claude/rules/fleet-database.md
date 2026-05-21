---
paths:
  - "server/datastore/**/*.go"
---

# Fleet Database Conventions

## Migration Files
- Location: `server/datastore/mysql/migrations/tables/`
- Naming: `YYYYMMDDHHMMSS_CamelCaseName.go` (timestamp + descriptive CamelCase)
- Every migration MUST have a corresponding `_test.go` file
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
