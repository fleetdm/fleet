---
name: update-data-dictionary
description: Compare recent database migrations against DATA-DICTIONARY.md and update it with any missing tables, columns, or changes. Use when asked to "update the data dictionary" or after adding migrations.
allowed-tools: Read, Grep, Glob, Edit, Bash(ls *), Bash(git log *)
effort: high
disable-model-invocation: true
---

# Update DATA-DICTIONARY.md

Compare recent migrations against the data dictionary and fix any discrepancies.

## Process

### 1. Find recent migrations

List migration files sorted by name (newest first):
```
ls -1 server/datastore/mysql/migrations/tables/*.go | grep -v _test | sort -r | head -30
```

### 2. Identify the last documented migration

Read the "Recent Schema Changes" section at the bottom of `DATA-DICTIONARY.md`. Find the most recent migration timestamp mentioned. All migrations after that timestamp need to be checked.

### 3. Read each undocumented migration

For each migration file not yet in the dictionary (skip `_test.go` files):
- Read the file
- Extract the DDL changes: `CREATE TABLE`, `ALTER TABLE ADD/DROP/MODIFY COLUMN`, `CREATE/DROP INDEX`, `RENAME TABLE`
- Classify as: new table, column addition, column removal, column modification, index change, table rename, or data-only migration

### 4. Check each change against DATA-DICTIONARY.md

For each DDL change:
- **New table**: check if the table has a section in the dictionary. If not, it needs one.
- **New column**: find the table's section and check if the column is documented.
- **Dropped column**: find the table's section and check if the column is still documented (it shouldn't be).
- **Table rename**: check if the dictionary uses the old or new name.
- **Index changes**: update the indexes section if the dictionary tracks them.
- **Data-only**: note in "Recent Schema Changes" but no table section updates needed.

### 5. Apply updates

For each discrepancy:
- Add missing table sections following the format of existing entries (Purpose, Key Fields, Relationships, Indexes, Usage Notes, Platform Affinity)
- Add missing columns to existing table sections in the right position
- Remove columns that were dropped
- Update table names that were renamed
- Add entries to the "Recent Schema Changes" tables (New Tables, Modified Tables, Data Changes)
- Update the "Last Updated" date at the top

### 6. Report

Summarize what was updated:
- Tables added
- Columns added/removed/modified
- Indexes changed
- Table renames
- Data-only migrations noted
