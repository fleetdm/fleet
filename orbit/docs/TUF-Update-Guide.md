# Pushing new releases to TUF

This document is a walkthrough guide for:
- A Fleet member to publish new updates to [Fleet's TUF service](tuf.fleetctl.com). See [Pushing updates](#pushing-updates).
- A Fleet member to delete targets from [Fleet's TUF service](tuf.fleetctl.com). See [Removing unused targets](#removing-unused-targets).
- A Fleet member to become a publisher of updates for [Fleet's TUF service](tuf.fleetctl.com). See [Becoming a new Fleet publisher](#becoming-a-new-fleet-publisher).

## Security

- The TUF keys for `targets`, `snapshot` and `timestamp` should be stored on a USB stick (used solely for this purpose). Whenever you need to push updates to Fleet's TUF repository you can temporarily copy the encrypted keys to your workstation (under the `keys/` folder, more on this below).
- The keys should be stored encrypted with its passphrase stored in 1Password (on a private vault).
- Every `fleetctl updates` command will prompt for the passphrases to decrypt the encrypted keys. You can input the passphrases every time or can alternatively set the following environment variables: `FLEET_TIMESTAMP_PASSPHRASE`, `FLEET_SNAPSHOT_PASSPHRASE` and `FLEET_TARGETS_PASSPHRASE`. Make sure to not leave traces of the passphrases (scripts, history and/or environment) when you are done.

## Syncing Fleet's TUF repository

> The `fleetctl updates` commands assume the folders `keys/`, `staged/` and `repository/` exist on the current working directory.

> IMPORTANT: When syncing the repository make sure to use `--exact-timestamps`. Otherwise `aws s3 sync` may not sync files that do not change in size, like `timestamp.json`.

- The `keys/` folder contains the encrypted private keys.
- The `staged/` folder contains uncommitted changes (usually empty because `fleetctl updates` commands automatically commit the changes).
- The `repository/` folder contains the full TUF repository.

Following are the commands to initialize the repository on your workstation:
```sh
mkdir /path/to/tuf.fleetctl.com

cd /path/to/tuf.fleetctl.com
mkdir -p ./repository
cp /Volumes/YOUR-USB-NAME/keys ./keys
mkdir -p ./staged

aws s3 sync s3://fleet-tuf-repo ./repository --exact-timestamps
```

## Pushing updates

> Before performing any actions on Fleet's TUF repository you must:
> 1. Make sure your local copy of the repository is up-to-date. See [Syncing Fleet's TUF repository](#syncing-fleets-tuf-repository).
> 2. Create a local backup in case we mess up with the repository:
>    ```sh
>    mkdir ~/tuf.fleetctl.com/backup
>    cp -r ~/tuf.fleetctl.com ~/tuf.fleetctl.com-backup
>    ```

### Releasing to the `edge` channel

> Make sure to install fleetd components using the `edge` channels in the three supported OSs (this is useful to smoke test the update).

Following is the list of components and each command for each operating system.

The commands show here update the local repository. After you are done running the commands below for each component, see [Pushing releases to Fleet's TUF repository](#pushing-releases-to-fleets-tuf-repository) to push the updates to Fleet's TUF repository (https://tuf.fleetctl.com).

#### orbit

The `orbit` executables are downloaded from the [GoReleaser Orbit action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-orbit.yaml).
Such action is triggered when git tagging a new orbit version with a tag of the form: `orbit-v1.15.0`.
    
> The following commands assume you are pushing version `1.15.0`.

```sh
# macOS
fleetctl updates add --target /path/to/downloaded/macos/orbit --platform macos --name orbit --version 1.15.0 -t edge
# Linux
fleetctl updates add --target /path/to/downloaded/linux/orbit --platform linux --name orbit --version 1.15.0 -t edge
# Windows
fleetctl updates add --target /path/to/downloaded/windows/orbit.exe --platform windows --name orbit --version 1.15.0 -t edge
```

#### desktop 

The Fleet Desktop executables are downloaded from the [Generate Fleet Desktop targets for Orbit action](https://github.com/fleetdm/fleet/actions/workflows/generate-desktop-targets.yml).
Such action is triggered by submitting a PR with the [following version string](https://github.com/fleetdm/fleet/blob/4a6bf0d447a2080f994da1e2f36ce6d51db88109/.github/workflows/generate-desktop-targets.yml#L27) changed.

> The following commands assume you are pushing version `1.15.0`.

```sh
# macOS
fleetctl updates add --target /path/to/macos/downloaded/desktop.app.tar.gz --platform macos --name desktop --version 1.15.0 -t edge
# Linux
fleetctl updates add --target /path/to/linux/downloaded/desktop.tar.gz --platform linux --name desktop --version 1.15.0 -t edge
# Windows
fleetctl updates add --target /path/to/windows/downloaded/fleet-desktop.exe --platform windows --name desktop --version 1.15.0 -t edge
```

#### swiftDialog

> macOS only component

The `swiftDialog` executable can be generated from a macOS host by running:
```sh
make swift-dialog-app-tar-gz version=2.2.1 build=4591 out-path=.
```

```sh
fleetctl updates add --target /path/to/macos/swiftDialog.app.tar.gz --platform macos --name swiftDialog --version 2.2.1 -t edge
```

#### nudge

> macOS only component

The `nudge` executable can be generated from a macOS host by running:
```sh
make nudge-app-tar-gz version=1.1.10.81462 out-path=.
```

```sh
fleetctl updates add --target /path/to/macos/nudge.app.tar.gz --platform macos --name nudge --version 1.1.10.81462 -t edge
```

#### osqueryd

Osquery executables are downloaded from the [Generate osqueryd targets for Fleetd action](https://github.com/fleetdm/fleet/blob/main/.github/workflows/generate-osqueryd-targets.yml).
Such action is triggered by submitting a PR with the [following version string](https://github.com/fleetdm/fleet/blob/7067ca586a4aa1a0377b387d4b4478a5958193ff/.github/workflows/generate-osqueryd-targets.yml#L27) changed.

> The following commands assume you are pushing version `5.9.1`.

```sh
# macOS
fleetctl updates add --target /path/to/downloaded/macos/osqueryd.app.tar.gz --platform macos-app --name osqueryd --version 5.9.1 -t edge
# Linux
fleetctl updates add --target /path/to/downloaded/linux/osqueryd --platform linux --name osqueryd --version 5.9.1 -t edge
# Windows
fleetctl updates add --target /path/to/downloaded/windows/osqueryd.exe --platform windows --name osqueryd --version 5.9.1 -t edge
```

#### Push updates

Once all components are updated in your local repository we need to push the changes to the remote repository.
See [Pushing releases to Fleet's TUF repository](#pushing-releases-to-fleets-tuf-repository).

### Promoting `edge` to the `stable` channel

> Make sure to install fleetd components using the `stable` channels in the three supported OSs (this is useful to smoke test the update).

Following is the list of components and each command for each operating system.

The commands show here update the local repository. After you are done running the commands below for each component, see [Pushing releases to Fleet's TUF repository](#pushing-releases-to-fleets-tuf-repository) to push the updates to Fleet's TUF repository (https://tuf.fleetctl.com).

#### orbit

> The following commands assume you are pushing version `1.15.0`.

```sh
# macOS
fleetctl updates add --target ./repository/targets/orbit/macos/edge/orbit --platform macos --name orbit --version 1.15.0 -t 1.15 -t 1 -t stable
# Linux
fleetctl updates add --target ./repository/targets/orbit/linux/edge/orbit --platform linux --name orbit --version 1.15.0 -t 1.15 -t 1 -t stable
# Windows
fleetctl updates add --target ./repository/targets/orbit/windows/edge/orbit.exe --platform windows --name orbit --version 1.15.0 -t 1.15 -t 1 -t stable
```

#### desktop

> The following commands assume you are pushing version `1.15.0`.

```sh
# macOS
fleetctl updates add --target ./repository/targets/desktop/macos/edge/desktop.app.tar.gz --platform macos --name desktop --version 1.15.0 -t 1.15 -t 1 -t stable
# Linux
fleetctl updates add --target ./repository/targets/desktop/linux/edge/desktop.tar.gz --platform linux --name desktop --version 1.15.0 -t 1.15 -t 1 -t stable
# Windows
fleetctl updates add --target ./repository/targets/desktop/windows/edge/fleet-desktop.exe --platform windows --name desktop --version 1.15.0 -t 1.15 -t 1 -t stable
```

#### swiftDialog

```sh
# macOS
fleetctl updates add --target ./repository/targets/swiftDialog/macos/edge/swiftDialog.app.tar.gz --platform macos --name swiftDialog --version 2.2.1 -t stable
```

#### nudge

```sh
# macOS
fleetctl updates add --target ./repository/targets/nudge/macos/edge/nudge.app.tar.gz --platform macos --name nudge --version 1.1.10.81462 -t stable
```

#### osqueryd

> The following commands assume you are pushing version `5.9.1`.

```sh
# macOS
fleetctl updates add --target ./repository/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz --platform macos-app --name osqueryd --version 5.9.1 -t 5.9 -t 5 -t stable
# Linux
fleetctl updates add --target ./repository/targets/osqueryd/linux/edge/osqueryd --platform linux --name osqueryd --version 5.9.1 -t 5.9 -t 5 -t stable
# Windows
fleetctl updates add --target ./repository/targets/osqueryd/windows/edge/osqueryd.exe --platform windows --name osqueryd --version 5.9.1 -t 5.9 -t 5 -t stable
```

#### Push updates

Once all components are updated in your local repository we need to push the changes to the remote repository.
See [Pushing releases to Fleet's TUF repository](#pushing-releases-to-fleets-tuf-repository).

### Pushing releases to Fleet's TUF repository

Once you are done with the changes on your local repository, you can use the following command to review the changes before pushing (`--dryrun` allows us to verify the upgrade before pushing):
```sh
AWS_PROFILE=tuf aws s3 sync ./repository s3://fleet-tuf-repo --dryrun
(dryrun) upload: repository/snapshot.json to s3://fleet-tuf-repo/snapshot.json
(dryrun) upload: repository/targets.json to s3://fleet-tuf-repo/targets.json
[...]
(dryrun) upload: repository/timestamp.json to s3://fleet-tuf-repo/timestamp.json
```

If all looks good, run the same command without the `--dryrun` flag.

> NOTE: Some things to note after the changes are pushed:
>   - Once pushed you might see some clients failing to upgrade due to some sha256 mismatches. These temporary failures are expected because it takes some time for caches to be invalidated (these errors should go away after a few minutes).
>   - The auto-update routines in orbit run every one hour, so you might need to wait up to an hour to verify hosts are auto-updating properly.

## Removing Unused Targets

If you've inadvertently published a target that is no longer in use, follow these steps to remove it.

> Before performing any actions on Fleet's TUF repository you must:
> 1. Make sure your local copy of the repository is up-to-date. See [Syncing Fleet's TUF repository](#syncing-fleets-tuf-repository).
> 2. Create a local backup in case we mess up with the repository:
>    ```sh
>    mkdir ~/tuf.fleetctl.com/backup
>    cp -r ~/tuf.fleetctl.com ~/tuf.fleetctl.com-backup
>    ```

1. You'll need the [`go-tuf`](https://github.com/theupdateframework/go-tuf) binary. The removal operations aren't integrated into `fleetctl` at the moment.
2. Use `tuf remove` to remove the target and update `targets.json`. Substitute `desktop/windows/stable/desktop.exe` with the target you intend to delete.
```sh
tuf remove desktop/windows/stable/desktop.exe
```
3. Snapshot, timestamp, and commit the changes.
```sh
tuf snapshot
tuf timestamp
tuf commit
```
4. Run the following command to generate a timestamp that expires in two weeks (otherwise the default expiration when using `go-tuf` commands is 1 day) 
```sh
fleetctl updates timestamp
```
5. Confirm that the version of the local `timestamp.json` file is more recent than that of the remote server.
6. Verify the changes that will be synced by running a dry sync. Include the `--delete` flag as you're removing targets.
```sh
aws s3 sync ./repository s3://fleet-tuf-repo --delete --dryrun
```
7. `diff` the local `targets.json` file with its remote version.
8. To upload the changes, perform a sync without the `--dryrun`:
```sh
aws s3 sync ./repository s3://fleet-tuf-repo --delete
```

## Becoming a New Fleet Publisher

> Before performing any actions on Fleet's TUF repository you must:
> 1. Make sure your local copy of the repository is up-to-date. See [Syncing Fleet's TUF repository](#syncing-fleets-tuf-repository).
> 2. Create a local backup in case we mess up with the repository:
>    ```sh
>    mkdir ~/tuf.fleetctl.com/backup
>    cp -r ~/tuf.fleetctl.com ~/tuf.fleetctl.com-backup
>    ```

### Generate targets+snapshot+timestamp keys

All commands shown in this guide are executed from `/path/to/tuf.fleetctl.com`:
```sh
cd /path/to/tuf.fleetctl.com
```

```sh
tuf gen-key targets
Enter targets keys passphrase:
Repeat targets keys passphrase:
Generated targets key with ID ae943cb8be8a849b37c66ed46bdd7e905ba3118c0c051a6ee3cd30625855a076
```
```sh
tuf gen-key snapshot
Enter snapshot keys passphrase:
Repeat snapshot keys passphrase:
Generated snapshot key with ID 1a4d9beb826d1ff4e036d757cfcd6e36d0f041e58d25f99ef3a20ae3f8dd71e3
```
```sh
tuf gen-key timestamp
Enter timestamp keys passphrase:
Repeat timestamp keys passphrase:
Generated timestamp key with ID d940df08b59b12c30f95622a05cc40164b78a11dd7d408395ee4f79773331b30
```

Share `staged/root.json` with Fleet member with the `root` role, who will sign with its root key and push to the repository.

### Root role signs the `staged/root.json`

Essentially the following commands are executed to sign the new keys:
- `tuf sign`
- `tuf snapshot`
- `tuf timestamp`
- `tuf commit`

## Misc issues

### Invalid timestamp.json version

The following issue was solved by resigning the timestamp metadata `fleetctl updates timestamp` (executed three times to increase the version to `4175`)
```sh
2022-08-23T13:44:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
2022-08-23T13:59:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
```