# MariaDB Support

Fleet primarily uses MySQL as its database, but also has some basic support for MariaDB >= 11.6.
Not all features may be supported, and there may some subtle bugs throughout the application.

## Quick Start

### Docker Services

The service names are database-agnostic:
- `database` - Main development database
- `database_test` - Test database
- `database_replica_test` - Replica test database

These services run MySQL by default, or MariaDB when using the override docker-compose-mariadb.yml file.

### Persistent Volumes

Data is stored in separate volumes to allow switching between databases:
- `mysql-persistent-volume` - MySQL data
- `mariadb-persistent-volume` - MariaDB data

This prevents data corruption and allows you to switch between MySQL and MariaDB without losing data.

### Starting services

#### MySQL (default)
```bash
docker compose up
```

### MariaDB
```bash
docker compose -f docker-compose.yml -f docker-compose-mariadb.yml up
```

## Schema Conversion

MariaDB has some SQL syntax differences from MySQL. The `fix-mariadb-schema.sh` script automatically converts the MySQL schema to be MariaDB-compatible.

### Usage

```bash
./tools/mariadb/fix-mariadb-schema.sh
```

This creates `server/datastore/mysql/schema-mariadb.sql`.
The migrations contained in `server/datastore/mysql/migrations/tables/` are not compatabile with mariadb syntax, so you cannot use the `prepare db` command you would use to typically. You need to create your development database with the `schema-mariadb.sql` file.
This `schema-mariadb.sql` file is also used when running tests. The schema file is selected by the test infrastructure based on the `FLEET_DB_CLIENT` environment variable.

It is important to note that there are data migrations in `server/datastore/mysql/migrations/data`. These are not included in the `schema-mariadb.sql` file. After you import the `schema-mariadb.sql` file into your mariaDB, you will still need to run `./build/fleet prepare db --dev`. This will run the data migrations and import data that is critical to having fleet run correctly.

## Tests with MariaDB

```bash
# Run tests
FLEET_DB_CLIENT=mariadb MYSQL_TEST=1 go test ./server/datastore/mysql/...
```