# TUF Update Guide

This document is a walkthrough guide for:
- A Fleet member to become a publisher of updates for [Fleet's TUF service](tuf.fleetctl.com).
- A Fleet member to publish new updates to [Fleet's TUF service](tuf.fleetctl.com). 

> The roles needed to push new updates are `targets`, `snapshot` and `timestamp`. See [Roles and Metadata](https://theupdateframework.io/metadata/).

## Become a New Fleet Publisher

### Security

- TUF keys for `targets`, `snapshot` and `timestamp` should be stored on a USB stick (used solely for this purpose).
- The keys are stored encrypted with a passphrase stored in 1Password (on a private vault).

### Sync Fleet's TUF repository

The `fleetctl updates --path=<SOME_PATH>` commands assume `keys/`, `staged/` and `repository/` are under
<SOME_PATH> (default value for <SOME_PATH> is the current directory `"."`).

```sh
cd /Volumes/FLEET-TUF/repository

cd /Volumes/FLEET-TUF
mkdir -p repository
mkdir -p keys
mkdir -p staged

aws s3 sync s3://fleet-tuf-repo ./repository
```

### Generate targets+snapshot+timestamp keys

All commands shown in this guide are executed from `/Volumes/FLEET-TUF`:
```sh
cd /Volumes/FLEET-TUF
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

## Pushing new updates

Following are tested steps to push new targets.

### 1. Backup current TUF repository

Just in case we break the remote TUF directory, let's do a local backup:
```sh
mkdir ~/tuf.fleetctl.com/backup
aws s3 sync s3://fleet-tuf-repo ~/tuf.fleetctl.com/backup
```

### 2. Make sure the local repository is up-to-date

```sh
aws s3 sync s3://fleet-tuf-repo ./repository
```

### 3. Setup Orbit in Linux, Windows, macOS

Install Orbit with the (to be updated) channels in the three supported OSs.

E.g. if we need to push a new version of `osqueryd` to `edge`, then generate and install a package with:
```sh
fleetctl package ... --osqueryd-channel=edge ...
```

### 4. Setup Orbit in one host that points to our repository

This allows us to verify that already running clients will upgrade successfully.

Serve current unmodified repository:
```sh
cd repository && python3 -m http.server
```

Generate packages using the local TUF server (in my case on a macOS host):
```sh
fleetctl package --type=pkg --update-url=http://localhost:8000 ...
[...]
```

Install generated `fleet-osquery.pkg`.

### 5. Actually pushing new updates

In this example we are promoting orbit from `edge` to `stable`:
```sh
./fleetctl-macos updates add --target ./repository/targets/orbit/linux/edge/orbit --platform linux --name orbit --version 1.1.0 -t 1.1 -t 1 -t stable
[...]
./fleetctl-macos updates add --target ./repository/targets/orbit/windows/edge/orbit.exe --platform windows --name orbit --version 1.1.0 -t 1.1 -t 1 -t stable
[...]
./fleetctl-macos updates add --target ./repository/targets/orbit/macos/edge/orbit --platform macos --name orbit --version 1.1.0 -t 1.1 -t 1 -t stable
[...]
```

`--dryrun` allows us to verify the upgrade before pushing:
```sh
AWS_PROFILE=tuf aws s3 sync ./repository s3://fleet-tuf-repo --dryrun
(dryrun) upload: repository/snapshot.json to s3://fleet-tuf-repo/snapshot.json
(dryrun) upload: repository/targets.json to s3://fleet-tuf-repo/targets.json
(dryrun) upload: repository/targets/orbit/linux/1.1.0/orbit to s3://fleet-tuf-repo/targets/orbit/linux/1.1.0/orbit
(dryrun) upload: repository/targets/orbit/linux/1.1/orbit to s3://fleet-tuf-repo/targets/orbit/linux/1.1/orbit
(dryrun) upload: repository/targets/orbit/linux/1/orbit to s3://fleet-tuf-repo/targets/orbit/linux/1/orbit
(dryrun) upload: repository/targets/orbit/linux/stable/orbit to s3://fleet-tuf-repo/targets/orbit/linux/stable/orbit
(dryrun) upload: repository/targets/orbit/macos/1.1.0/orbit to s3://fleet-tuf-repo/targets/orbit/macos/1.1.0/orbit
(dryrun) upload: repository/targets/orbit/macos/1.1/orbit to s3://fleet-tuf-repo/targets/orbit/macos/1.1/orbit
(dryrun) upload: repository/targets/orbit/macos/1/orbit to s3://fleet-tuf-repo/targets/orbit/macos/1/orbit
(dryrun) upload: repository/targets/orbit/macos/stable/orbit to s3://fleet-tuf-repo/targets/orbit/macos/stable/orbit
(dryrun) upload: repository/targets/orbit/windows/1.1.0/orbit.exe to s3://fleet-tuf-repo/targets/orbit/windows/1.1.0/orbit.exe
(dryrun) upload: repository/targets/orbit/windows/1.1/orbit.exe to s3://fleet-tuf-repo/targets/orbit/windows/1.1/orbit.exe
(dryrun) upload: repository/targets/orbit/windows/1/orbit.exe to s3://fleet-tuf-repo/targets/orbit/windows/1/orbit.exe
(dryrun) upload: repository/targets/orbit/windows/stable/orbit.exe to s3://fleet-tuf-repo/targets/orbit/windows/stable/orbit.exe
(dryrun) upload: repository/timestamp.json to s3://fleet-tuf-repo/timestamp.json
```

In this other example we are updating osquery's `edge` channel:

```sh
./fleetctl-macos updates add --target /Users/luk/Downloads/tuf-osqueryd/osqueryd.exe --platform windows --name osqueryd --version 5.5.1 -t edge
[...]
./fleetctl-macos updates add --target /Users/luk/Downloads/tuf-osqueryd/osqueryd.app.tar.gz --platform macos-app --name osqueryd --version 5.5.1 -t edge
[...]
./fleetctl-macos updates add --target /Users/luk/Downloads/tuf-osqueryd/osqueryd --platform linux --name osqueryd --version 5.5.1 -t edge
[...]
```

`--dryrun` allows us to verify the upgrade before pushing:
```sh
aws s3 sync ./repository s3://fleet-tuf-repo --profile tuf --dryrun
(dryrun) upload: repository/snapshot.json to s3://fleet-tuf-repo/snapshot.json
(dryrun) upload: repository/targets.json to s3://fleet-tuf-repo/targets.json
(dryrun) upload: repository/targets/osqueryd/linux/5.5.1/osqueryd to s3://fleet-tuf-repo/targets/osqueryd/linux/5.5.1/osqueryd
(dryrun) upload: repository/targets/osqueryd/linux/edge/osqueryd to s3://fleet-tuf-repo/targets/osqueryd/linux/edge/osqueryd
(dryrun) upload: repository/targets/osqueryd/macos-app/5.5.1/osqueryd.app.tar.gz to s3://fleet-tuf-repo/targets/osqueryd/macos-app/5.5.1/osqueryd.app.tar.gz
(dryrun) upload: repository/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz to s3://fleet-tuf-repo/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz
(dryrun) upload: repository/targets/osqueryd/windows/5.5.1/osqueryd.exe to s3://fleet-tuf-repo/targets/osqueryd/windows/5.5.1/osqueryd.exe
(dryrun) upload: repository/targets/osqueryd/windows/edge/osqueryd.exe to s3://fleet-tuf-repo/targets/osqueryd/windows/edge/osqueryd.exe
(dryrun) upload: repository/timestamp.json to s3://fleet-tuf-repo/timestamp.json
```

### 6. Verify the already running test host

Verify host enrolled in step (4) upgraded to the new versions successfully.

### 7. Verify generation of new packages

```sh
fleetctl package --type=pkg --update-url=http://localhost:8000 ...
fleetctl package --type=msi --update-url=http://localhost:8000 ...
fleetctl package --type=deb --update-url=http://localhost:8000 ...
```

### 8. Push!

Run the same command shown above, but without `--dryrun`
```sh
aws s3 sync ./repository s3://fleet-tuf-repo --profile tuf
```

### 9. Final Verification

Now that the repository is pushed, verify that the hosts enrolled in step (3) update as expected. 

### Removing Unused Targets

If you've inadvertently published a target that is no longer in use, follow these steps to remove it.

### 1. Preparation

1. Backup the remote TUF directory to your local machine.

```sh
mkdir ~/tuf.fleetctl.com/backup
aws s3 sync s3://fleet-tuf-repo ~/tuf.fleetctl.com/backup
```

2. Ensure your local repository mirrors the current state.

```sh
aws s3 sync s3://fleet-tuf-repo ./repository
```

3. You'll need the [`go-tuf`](https://github.com/theupdateframework/go-tuf) binary. The removal operations aren't integrated into `fleetctl` at the moment.

### 2. Local Target Removal

1. Use `tuf remove` to remove the target and update `targets.json`. Substitute `desktop/windows/stable/desktop.exe` with the target you intend to delete.

```sh
tuf remove desktop/windows/stable/desktop.exe
```

2. Snapshot, timestamp, and commit the changes.

```sh
tuf snapshot
tuf timestamp
tuf commit
```

### 3. Verification Before Publishing

1. Confirm that the version of the local `timestamp.json` file is more recent than that of the remote server.

2. Verify the changes that will be synced by running a dry sync. Include the `--delete` flag as you're removing targets.

```sh
aws s3 sync ./repository s3://fleet-tuf-repo --delete --dryrun
```

3. `diff` the local `targets.json` file with its remote version.

### 4. Publish and Confirm

1. To upload the changes, perform a sync without the `--dryrun`.

```sh
aws s3 sync ./repository s3://fleet-tuf-repo --delete
```

2. Check the changes on the remote repository.

### Issues found

#### Invalid timestamp.json version

The following issue was solved by resigning the timestamp metadata `fleetctl updates timestamp` (executed three times to increase the version to 4175)
```sh
2022-08-23T13:44:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
2022-08-23T13:59:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
```

## Notes

- "Measure thrice cut once": Steps 3, 4, 5 and 6 allows us to verify the repository is in good shape before pushing to [Fleet's TUF service](tuf.fleetctl.com).
- Steps may look different if the upgrade is performed on a Linux or Windows host.
