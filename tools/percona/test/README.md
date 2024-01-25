# Test migrations with Percona Server XtraDB 5.7.25

> IMPORTANT: 
> - The test performed here will clear your local database (make sure to run `make db-backup` before running this, and run `make db-restore` after your are done).
> - This test was developed and tested on a macOS Intel device.

Following are the instructions to test Fleet DB migrations with a specific version of Percona Server XtraDB (5.7.25).

Dependencies:
- Docker for Mac.
- `mysql` client (`brew install mysql-client`).

1. At the root of the repository run:
```sh
./tools/percona/test/upgrade.sh
```
2. Once the script finishes (you should see `Migrations completed.` at the very end), run `fleet serve` and perform smoke tests as usual.
