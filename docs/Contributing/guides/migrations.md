# Fleet database migrations

- [Adding/Updating tables](#addingupdating-tables)
- [Populating the database with default data](#populating-the-database-with-default-data)


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

## Populating the database with default data

Note: This pattern is now deprecated, new data changes are done using the same migrations process as for tables. Since there are a few data migrations using this obsolete pattern, we keep its documentation here:

Populating built-in data is also performed through migrations. All table migrations are performed before any data migrations.

Note: Data migrations can be mutable. If tables are altered in a way that would render a data migration invalid (columns changed/removed), data migrations should be updated to comply with the new schema. Data migrations will not be re-run when they have already been run against a database, but they must be updated to maintain compatibility with a fresh DB.

From the project root, run the following shell command:

``` bash
make migration name=NameOfMigration
```

Move the migration file from [server/datastore/mysql/migrations/tables/](https://github.com/fleetdm/fleet/tree/97b4d1f3fb30f7b25991412c0b40327f93cb118c/server/datastore/mysql/migrations/tables) to [server/datastore/mysql/migrations/data/](https://github.com/fleetdm/fleet/tree/97b4d1f3fb30f7b25991412c0b40327f93cb118c/server/datastore/mysql/migrations/data), and change the `package tables` to `package data`.

Proceed as for table migrations, editing and running the newly created migration file.

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Learn about creating and updating database migrations for Fleet.">
