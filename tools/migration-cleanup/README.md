# Migration cleanup

`migration-cleanup` repairs Fleet's MySQL migration status tables after a release
candidate branch renumbers migration files that may already have run in an
existing database.

The normal target is the RC branch that contains the final, renumbered migration
filenames, such as `rc-minor-fleet-v4.86.0` or `rc-patch-fleet-v4.73.2`. The
tool compares that branch to `main`, finds migration file renames in
`server/datastore/mysql/migrations/tables` and
`server/datastore/mysql/migrations/data`, and generates SQL for the matching
`migration_status_tables` or `migration_status_data` rows.

## When to use it

Use this after a database has already run migrations from an RC build and later
Fleet cannot complete startup or `fleet prepare db` because the same migrations
were renumbered on the RC branch.

The tool only updates Fleet's migration status tables. It does not run, roll
back, or modify migration files.

Before applying against a shared or production database:

- take a database backup
- run against the writer database
- run `--dry-run` first
- inspect the generated SQL

## Build and help

Run these commands from the root of a `fleetdm/fleet` checkout. If you run the
tool from somewhere else, pass `-c /path/to/fleet` so it can inspect the correct
git history.

```sh
go run ./tools/migration-cleanup --help
go build -o build/migration-cleanup ./tools/migration-cleanup
```

## Generate SQL

SQL generation is the default mode and does not connect to MySQL.

```sh
go run ./tools/migration-cleanup -b rc-minor-fleet-v4.86.0
```

Write the SQL to a file:

```sh
go run ./tools/migration-cleanup \
  -b rc-minor-fleet-v4.86.0 \
  -o migration-cleanup.sql
```

Use `-c` when running the tool from a different working tree:

```sh
go run ./tools/migration-cleanup \
  -c /path/to/fleet \
  -b rc-minor-fleet-v4.86.0
```

If the branch only exists on the remote, pass the plain branch name. The tool
tries the local branch first, then `origin/<branch>`.

## Dry run

Dry run connects to MySQL, loads the real rows from `migration_status_tables`
and `migration_status_data`, simulates the generated SQL, and reports whether
the final state would be valid. It does not execute the generated SQL.

```sh
go run ./tools/migration-cleanup \
  -b rc-minor-fleet-v4.86.0 \
  --dry-run \
  --db-host 127.0.0.1 \
  --db-user fleet \
  --db-password insecure \
  --db-name fleet
```

You can also use Fleet MySQL config flags or a Fleet config file:

```sh
go run ./tools/migration-cleanup \
  -b rc-minor-fleet-v4.86.0 \
  --dry-run \
  --config fleet.yml
```

## Apply

Apply runs the same generated SQL in a transaction.

```sh
go run ./tools/migration-cleanup \
  -b rc-minor-fleet-v4.86.0 \
  --apply \
  --db-host 127.0.0.1 \
  --db-user fleet \
  --db-password insecure \
  --db-name fleet
```

`--dry-run` and `--apply` are mutually exclusive.

## Example: rc-minor-fleet-v4.86.0

This was the move-up renumber case tested against a local MySQL database that
had already run the pre-renumber 4.86 RC migrations. Running the latest RC
binary against that database failed while rerunning a migration that had already
been applied under its old version ID.

Generate SQL:

```sh
go run ./tools/migration-cleanup -b rc-minor-fleet-v4.86.0
```

Detected renumbers:

```text
Found 11 migration renumber(s):
  [tables] 20260427134220 -> 20260522195224
  [tables] 20260428125634 -> 20260522195225
  [tables] 20260429180725 -> 20260522195226
  [tables] 20260430103635 -> 20260522195227
  [tables] 20260506132626 -> 20260522195229
  [tables] 20260506171058 -> 20260522195230
  [tables] 20260512143542 -> 20260522195231
  [tables] 20260512173249 -> 20260522195232
  [tables] 20260512173250 -> 20260522195233
  [tables] 20260518124441 -> 20260522195234
  [tables] 20260518150028 -> 20260522195235
```

The dry run for the broken local database reported the real duplicate row and
the id shifts the generated SQL would perform:

```text
migration_status_tables: duplicate version_id=20260522195224; would keep id=518, delete ids=[530]
migration_status_tables: would make space by shifting 0 row(s) after version_id=20260522195235 by +12
migration_status_tables: would shift 11 row(s) by +12
Dry-run: SQL will apply cleanly.
```

After `--apply`, rerunning the latest 4.86 RC Fleet migrations reported that
migrations were already complete.

## Example: rc-patch-fleet-v4.73.2

This was the move-down renumber style case used to verify the generated SQL for
older patch RCs.

```sh
go run ./tools/migration-cleanup -b rc-patch-fleet-v4.73.2
```

Detected renumbers:

```text
Found 2 migration renumber(s):
  [tables] 20250904115553 -> 20250816115553
  [tables] 20250918154557 -> 20250817154557
```

For this shape, the generated SQL remaps those version IDs, removes duplicate
status rows if present, computes the id increment from the current table state,
and shifts affected rows with `ORDER BY id DESC`. The generated "make space"
update may affect zero rows; that is expected when there are no later rows to
move.

## Notes

- `--branch` should point at the branch with the final migration filenames.
- The generated SQL is the source of truth for SQL output, dry-run simulation,
  and apply mode.
- Dry run validates against real table data, but apply should still be preceded
  by a database backup.
- The tool is intentionally scoped to grouped migration renumbering on an RC
  branch. It is not a general-purpose migration status editor.
