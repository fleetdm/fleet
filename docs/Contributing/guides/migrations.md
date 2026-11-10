# Fleet database migrations

- [Adding/Updating tables](#addingupdating-tables)
- [The Fleet 5 baseline](#the-fleet-5-baseline)


## Adding/updating tables

We manage database schemas by a series of migrations defined in go code. We use a customized version of the Goose migrations tool to handle these migrations.

Note: Table migrations should be considered immutable once committed to the Fleet repo. Any changes to an existing table should occur in a new migration executing ALTERs.

Also, note that we don't use the `Down` part of the migrations anymore (older migrations did implement them). The `Down` function should just `return nil`, as we use forward-only migrations.

From the project root, run the following shell command:

``` bash
make migration name=NameOfMigration
```

Now edit the generated migration file in [server/datastore/mysql/migrations/tables/](https://github.com/fleetdm/fleet/tree/97b4d1f3fb30f7b25991412c0b40327f93cb118c/server/datastore/mysql/migrations/tables).

You can then update the database by running the following shell commands:

``` bash
make fleet
./build/fleet prepare db
```

## The Fleet 5 baseline

The first migration, `20261103120000_Fleet5Baseline.go`, creates the whole Fleet 5 schema and its seed data (built-in labels, default app config, etc.) from `baseline_schema.sql`. It replaced every Fleet 4 table and data migration. Never edit the baseline: add a new migration instead, including for default data.

Fleet 4 databases must be upgraded to the latest Fleet 4.95.x before upgrading to Fleet 5. A 4.95.x database already has the baseline schema, so on those the baseline only checks that every migration in `Fleet4Migrations` (`fleet4_migrations.go`) is applied, then records itself. It refuses any other Fleet 4 database. The Fleet 4 migration records are kept, and the migration status check treats them as known.

A migration added in a 4.95.x patch release must also be merged, unchanged, into Fleet 5. Its version is newer than the baseline, so goose runs it after the baseline on fresh installs and on databases upgraded from an earlier 4.95.x, and skips it on databases that already applied it. Don't add it to `Fleet4Migrations`.

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Learn about creating and updating database migrations for Fleet.">
