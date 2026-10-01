# Migration list

`migration-list.sh` prints the most recent table migrations sorted by version ID, each labeled with the earliest release tag or branch that includes it. Use it to catch migrations numbered out of order across releases. Fleet only runs migrations newer than the last one a database applied, so a migration in an older release must be numbered before every migration that is only in a newer release or `main`.

It only reads local git data, so run `git fetch origin --tags` first, or pass `-f`.

```sh
./tools/migration-list/migration-list.sh
```

```
version id | earliest release tag or branch that includes this migration
20260923180303 - 4.93.0
20260925155809 - 4.93.0
20260925160000 - 4.93.0
20260925180001 - main    <--
```

The script compares the most recent releases oldest first, then `origin/main`, and labels each migration with the first one that has it. Releases are `fleet-vX.Y.Z` tags, or `origin/rc-minor-fleet-vX.Y.Z` and `origin/rc-patch-fleet-vX.Y.Z` branches for versions that aren't tagged yet, labeled with their version. A row is marked with `<--` when its label is different from the row above it, skipping rows marked `<-- NEW`.

To check a branch, pass `-b` with the branch it will merge into. This also compares `HEAD`, so migrations only in `HEAD` are labeled `HEAD`, and the ones it added or renamed are marked with `<-- NEW`. For a branch that targets an RC, the list also includes the migrations on `origin/main`, so a cherry-picked migration that is newer than migrations only on `main` shows up between `main` rows:

```sh
./tools/migration-list/migration-list.sh -b origin/rc-minor-fleet-v4.93.0
```

```
20260923180303 - 4.93.0  <--
20260925155809 - 4.93.0
20260925160000 - 4.93.0
20260925182107 - main    <--
20260925182113 - main
20260926120000 - HEAD    <-- NEW
20260928054420 - main
```

Run with `-h` for all options.
