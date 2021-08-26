# Fleet Database migrations

- [Adding/Updating tables](#addingupdating-tables)
- [Populating the database with default data](#populating-the-database-with-default-data)


## Adding/Updating tables

Database schemas are managed by a series of migrations defined in go code. We use a customized version of the Goose migrations tool to handle these migrations.

Note: Once committed to the Fleet repo, table migrations should be considered immutable. Any changes to an existing table should take place in a new migration executing ALTERs.

Also note that we don't use the `Down` part of the migrations anymore (older migrations did implement them). The `Down` function should just `return nil`, as we use forward-only migrations.

From the project root run the following shell command:

``` bash
make migration name=NameOfMigration
```

Now edit the generated migration file in [server/datastore/mysql/migrations/tables/](../../server/datastore/mysql/migrations/tables/).

You can then update the database by running the following shell commands:

``` bash
make fleet
./build/fleet prepare db
```

## Populating the database with default data

Note: This pattern will soon be changing. Please check with @zwass if you think you need to write a data migration.

Populating built in data is also performed through migrations. All table migrations are performed before any data migrations.

Note: Data migrations can be mutable. If tables are altered in a way that would render a data migration invalid (columns changed/removed), data migrations should be updated to comply with the new schema. Data migrations will not be re-run when they have already been run against a database, but they must be updated to maintain compatibility with a fresh DB.

From the project root run the following shell command:

``` bash
make migration name=NameOfMigration
```

Move the migration file from [server/datastore/mysql/migrations/tables/](../../server/datastore/mysql/migrations/tables/) to [server/datastore/mysql/migrations/data/](../../server/datastore/mysql/migrations/data/), and change the `package tables` to `package data`.

Proceed as for table migrations, editing and running the newly created migration file.


