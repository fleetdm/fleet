# Migration list

`migration-list.sh` prints the most recent table migrations sorted by version ID, each labeled with the earliest Fleet release that includes it. Use it to check that migrations from a release run before migrations that are only on `main`, for example after a cherry-pick into a release candidate.

It only reads local git data, so run `git fetch origin --tags` first, or pass `-f`.

```sh
./tools/migration-list/migration-list.sh
```

```
version id | earliest fleet version this migration is included in
20260923180303 - 4.93.0
20260925155809 - 4.93.0
20260925160000 - 4.93.0
20260925182107 - main    <--
```

Releases are `fleet-vX.Y.Z` tags, or `origin/rc-minor-fleet-vX.Y.Z` and `origin/rc-patch-fleet-vX.Y.Z` branches for versions that aren't tagged yet. A row is marked with `<--` when its release is different from the row above it.

To check a branch, pass `-b` with the branch it will merge into. This compares `HEAD` instead of `origin/main` and marks migrations the branch added or renamed with `<-- NEW`:

```sh
./tools/migration-list/migration-list.sh -b origin/main
```

Run with `-h` for all options.
