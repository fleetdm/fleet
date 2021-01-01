# Fleet Database migrations

- [Adding/Updating tables](#addingupdating-tables)
- [Populating the database with default data](#populating-the-database-with-default-data)


## Adding/Updating tables

Database schemas are managed by a series of migrations defined in go code. We use a customized version of the Goose migrations tool to handle these migrations.

Note: Once committed to the Fleet repo, table migrations should be considered immutable. Any changes to an existing table should take place in a new migration executing ALTERs.

From the project root run the following shell commands:

``` bash
go get github.com/kolide/goose
cd server/datastore/mysql/migrations/tables
goose create AddColumnFooToUsers
```

Find the file you created in the migrations directory and edit it:

* delete the import line for goose: `github.com/pressly/goose`
* change `goose.AddMigration(...)` to `MigrationClient.AddMigration(...)`
* add your migration code

You can then update the database by running the following shell commands:

``` bash
make build
build/fleet prepare db
```

## Populating the database with default data

Populating built in data is also performed through migrations. All table migrations are performed before any data migrations.

Note: Data migrations can be mutable. If tables are altered in a way that would render a data migration invalid (columns changed/removed), data migrations should be updated to comply with the new schema. Data migrations will not be re-run when they have already been run against a database, but they must be updated to maintain compatibility with a fresh DB.

From the project root run the following shell commands:

``` bash
go get github.com/kolide/goose
cd server/datastore/mysql/migrations/data
goose create PopulateFoo
```

Proceed as for table migrations, editing and running the newly created migration file.


