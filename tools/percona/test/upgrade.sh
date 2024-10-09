#!/bin/bash

set -x
set -e

# Up to `fleet-v4.40.0` there are no migration issues with Percona Server XtraDB's `pxc_strict_mode=ENFORCING` default value.
# We introduced issues with `pxc_strict_mode=ENFORCING` in DB migrations in `fleet-v4.41.0` and in `fleet-v4.42.0`.

# Bring everything down.
docker compose down
docker volume rm fleet_mysql-persistent-volume

# Start dependencies using Percona XtraDB as MySQL server.
# NOTE: To troubleshoot, remove `>/dev/null`.
FLEET_MYSQL_IMAGE=percona/percona-xtradb-cluster:8.0.36 docker compose up >/dev/null 2>&1 &

export MYSQL_PWD=toor

until mysql --host 127.0.0.1 --port 3306 -uroot -e 'SELECT 1=1;' ; do
    echo "Waiting for Percona XtraDB MySQL Server..."
    sleep 10
done
echo "Percona XtraDB MySQL Server is up and running, continuing..."

# Checkout and build `fleet-4.42.0`.
git checkout fleet-v4.42.0
make generate && make fleet

# Set pxc_strict_mode=PERMISSIVE to run migrations up to fleet-v4.42.0,
# which was the last migration released with `pxc_strict_mode=ENFORCING` issues.
mysql --host 127.0.0.1 --port 3306 -uroot -e 'SET GLOBAL pxc_strict_mode=PERMISSIVE;'

# Run migrations up to fleet-v4.42.0.
make db-reset

# Set `pxc_strict_mode` back to the `ENFORCING` default.
mysql --host 127.0.0.1 --port 3306 -uroot -e 'SET GLOBAL pxc_strict_mode=ENFORCING;'

# Run migrations from fleet-v4.42.0 up to latest to catch any future bugs when running with `pxc_strict_mode=ENFORCING`.
git checkout main
make generate && make fleet
./build/fleet prepare db --dev --logging_debug
