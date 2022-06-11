# MySQL Replica Testing

This directory contains scripts to run/test a local Fleet instance with a MySQL Read Replica.

## Run MySQL Main and Read Replica Docker Images

> Run all commands from fleet's root repository directory.

```sh
docker-compose -f ./tools/mysql-replica-testing/docker-compose.yml up
```

## Configure MySQL Main and Read Replica

```sh
# Configure the main and read replica for replication.
make db-replica-setup

# Reset the main database.
make db-replica-reset
```

## Run Fleet with Read Replica

```sh
make db-replica-run
```
