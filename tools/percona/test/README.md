# Test migrations with Percona Server XtraDB 5.7.25

Following are the instructions to test Fleet DB migrations with a specific version of Percona Server XtraDB (5.7.25).

> The following was tested on a macOS (Intel) device.

1. At the root of the repository run:
```sh
./tools/percona/test/upgrade.sh
```
2. Once the script finishes (you should see `Migrations completed.` at the very end), run `fleet serve` and perform smoke tests as usual.
