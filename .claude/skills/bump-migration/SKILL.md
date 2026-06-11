---
name: bump-migration
description: Bump a database migration's timestamp to the current time. Required when a PR's migration is older than one already merged to main. Use when asked to "bump migration", "update migration timestamp", or when a migration ordering conflict is detected.
allowed-tools: Bash(go run *), Bash(make dump-test-schema*), Bash(git diff*), Bash(ls *), Read, Grep, Glob
model: sonnet
effort: medium
---

# Bump a database migration timestamp

Bump the migration: $ARGUMENTS

## When to use

This is required when a PR has a database migration with a timestamp older than a migration already merged to main. This happens when a PR has been pending merge for a while and another PR got merged with a more recent migration.

## Process

### 1. Identify the migration to bump

If the user provided a filename, use that. Otherwise, find migrations on this branch that are older than the latest on main:

```bash
# List migrations on this branch that aren't on main
git diff origin/main --name-only -- server/datastore/mysql/migrations/tables/
```

### 2. Run the bump tool

The tool lives at `tools/bump-migration/main.go`. Run it from the repo root:

```bash
go run tools/bump-migration/main.go --source-migration YYYYMMDDHHMMSS_MigrationName.go
```

This will:
- Rename the migration file with a new current timestamp
- Rename the test file (if it exists)
- Update all function names inside both files (`Up_OLDTS` → `Up_NEWTS`, `Down_OLDTS` → `Down_NEWTS`, `TestUp_OLDTS` → `TestUp_NEWTS`)

### 3. Optionally regenerate the schema

If the migration affects the schema, add `--regen-schema` to also run `make dump-test-schema`:

```bash
go run tools/bump-migration/main.go --source-migration YYYYMMDDHHMMSS_MigrationName.go --regen-schema
```

### 4. Verify

- Check that the old files are gone and new files exist with the updated timestamp
- Verify the function names inside the files match the new timestamp
- Run `go build ./server/datastore/mysql/migrations/...` to check compilation

## Rules
- Always run from the repo root
- Provide the migration filename, not the test filename
- The tool handles both the migration and its test file automatically
