# Test migrations with Percona Server XtraDB 5.7.25

> IMPORTANT: 
> - The test performed here will clear your local database.
> - This test was developed and tested on a macOS Intel device.

Following are the instructions to test Fleet DB migrations with a specific version of Percona Server XtraDB (5.7.25). We need to run this specific test for users running this specific version of Percona Server.
The test will run migrations with `pxc_strict_mode=PERMISSIVE` up until `fleet-v4.42.0` and then run the remaining migrations with `pxc_strict_mode=ENFORCING` (default) because starting in `fleet-v4.44.0` we will attempt to make every migration compatible with running this specific version of Percona Server with the default setting.

Dependencies:
- Docker for Mac.
- `mysql` client (`brew install mysql-client`).

Everything should be executed at the root of the repository.

1. Backup first by running `make db-backup`.
1. Make sure to be on latest `main`:
```sh
git checkout main
git pull origin main
```
1. Run the upgrade test script: `./tools/percona/test/upgrade.sh`.
1. Once the script finishes (you should see `Migrations completed.` at the very end), run `fleet serve` and perform smoke tests as usual.
1. Restore your previous setup by running the following:
```sh
docker compose down
docker volume rm fleet_mysql-persistent-volume
docker compose up
make db-restore
```