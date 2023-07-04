# Releasing Orbit

What we call "Orbit" is actually a group of components:
1. Orbit executable: Orbit is the director of the orchestra. It manages itself all the other components.
2. Osquery executable/bundle.
3. "Fleet Desktop" application: Renders Fleet's tray icon on the user desktop session and provides transparency to the end-user about what Fleet collects from the device.

# Auto-update

Orbit runs an auto-updater routine that polls a [TUF](https://theupdateframework.io/) server for new
updates in any of the three
components mentioned above. Each component (also known as "target") can be updated independently. This document aims to
describe all the steps needed to release a new version of each target.

## Methodology

Our TUF server provides two channels, `edge` and `stable`.
- `stable` is what all users in production use.
- `edge` is used to verify updates before releasing to `stable`.

At a high level, the following steps are used to release updates:

1. The new update/s are first pushed to the `edge` channel.
2. Smoke testing is used to verify the updates are working as expected.
3. The new update/s are then pushed to the `stable` channel.
4. Same as (2), smoke testing is used to verify the updates are working as expected.

# Actors

In all the steps described herein, there are two actors:
- Team member "Updater" pushing the updates. This actor requires:
    1. Authorized signing keys for pushing new updates on the TUF repository.
    2. Write access to the TUF server (to update https://tuf.fleetctl.com).
    3. Write privileges to the [fleet](https://github.com/fleetdm/fleet) repository to create pull
       requests and trigger Github Actions.
- Team member "Verifier" verifying/testing the pushed updates.

The majority of the steps are run by the "Updater" team member, unless stated otherwise.

# Updating Orbit

## 1. Edge Release

### Setup

The Verifier will setup a CentOS, Ubuntu, Windows and macOS hosts with Orbit running from the `edge` channel:
```sh
fleetctl package --type=pkg --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --orbit-channel edge
fleetctl package --type=msi --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --orbit-channel edge
fleetctl package --type=deb --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --orbit-channel edge
fleetctl package --type=rpm --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --orbit-channel edge
```

> NOTE: The fleetctl version to use should be the latest released version, installed via `sudo npm install -g fleetctl`.

### Steps

Assuming version `vX.Y.Z` is being released.

1. Run `make changelog-orbit` to generate the `orbit/CHANGELOG.md` changes for Orbit.
2. Edit `orbit/CHANGELOG.md` accordingly.
3. Checkout a new branch (we generally use `prepare-orbit-vX.Y.Z`), commit the changes and tag the repository:
```sh
git checkout -b prepare-orbit-vX.Y.Z
git add -u
git commit -m "Prepare changes for Orbit vX.Y.Z"
git tag orbit-vX.Y.Z
```
4. Push the branch and the tag:
```sh
git push origin prepare-orbit-vX.Y.Z
git push origin --tags
```
5. After pushing the branch, create a pull request.
6. The pushed tag will trigger a new build of the following Github Action:
   [goreleaser-orbit.yaml](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-orbit.yaml).
7. If the above Github Action ran successfully then a new "DRAFT" release will be created for Orbit
   vX.Y.Z: https://github.com/fleetdm/fleet/releases.
8. Download and extract the assets (one for each platform Orbit supports).
9. Push the downloaded+extracted assets to the `edge` channel on our TUF repository (https://tuf.fleetctl.com/):
```sh
# Having extracted the asset for Linux in `./orbit-linux`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target orbit \
    --platform linux \
    --name ./orbit-linux \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Linux in `./orbit-darwin`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target orbit \
    --platform macos \
    --name ./orbit-darwin \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Windows in `./orbit.exe`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target orbit \
    --platform windows \
    --name ./orbit.exe \
    --version X.Y.Z -t X.Y -t X -t edge
```

### Verification

Verifier will make sure all the hosts have updated the target successfully. The update interval delay can be up to 15 minutes.
Verifier can run `SELECT * from orbit_info;` live query on the hosts, which will provide the orbit version (confirming the update was successful).

Once orbit has auto-updated on all hosts, Verifier runs the usual smoke testing on the 4 OSs (e.g. refetching & live querying hosts, listing software, etc.).

## 2. Stable Release

### Setup

Verifier runs the same setup as `edge`, but without setting the `--orbit-channel` flag (the default value is `stable`).

### Steps

Run the same `fleetctl updates add` command as the `edge` case with the same targets, but with `-t stable`.

### Verification

Verification is the same as with the `edge` case.

# Updating Osquery

## 1. Edge Release

### Setup

The Verifier will setup a CentOS, Ubuntu, Windows and macOS host with `osqueryd` that uses the `edge` channel:
```sh
fleetctl package --type=pkg --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --osqueryd-channel edge
fleetctl package --type=msi --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --osqueryd-channel edge
fleetctl package --type=deb --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --osqueryd-channel edge
fleetctl package --type=rpm --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --osqueryd-channel edge
```

> NOTE: The fleetctl version to use should be the latest released version, installed via `sudo npm install -g fleetctl`.

### Steps

Assuming version `vX.Y.Z` is being released.

1. Checkout a branch and edit the `OSQUERY_VERSION` env variable in `.github/workflows/generate-osqueryd-targets.yml`.
2. Push and create a pull request.
3. Once the pull request is created a Github Action will be triggered
   [generate-osqueryd-targets.yml](https://github.com/fleetdm/fleet/actions/workflows/generate-osqueryd-targets.yml).
   It generates the osqueryd targets for macOS, Windows and Linux as artifacts.
4. Download the artifacts from the previous step and push them to the `edge` channel:
```sh
# Having extracted the asset for Linux in `./osqueryd`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target osqueryd \
    --platform linux \
    --name ./osqueryd \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Linux in `./osqueryd.app.tar.gz`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target osqueryd \
    --platform macos-app \
    --name ./osqueryd.app.tar.gz \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Windows in `./osqueryd.exe`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target osqueryd \
    --platform windows \
    --name ./osqueryd.exe \
    --version X.Y.Z -t X.Y -t X -t edge
```

### Verification

Verifier will make sure all the hosts have updated the target successfully. The update interval delay can be up to 15 minutes.
Verifier can run `SELECT * from osquery_info;` live query on the hosts, which will provide the osquery version (confirming the update was successful).

Once osqueryd has auto-updated on all hosts, Verifier runs the usual smoke testing on the 4 OSs (e.g. refetching & live querying hosts, listing software, etc.).

## 2. Stable Release

### Setup

Verifier runs the same setup as `edge`, but without setting the `--osqueryd-channel` flag (the default value is `stable`).

### Steps

Run the same `fleetctl updates add` command as the `edge` case with the same targets, but with `-t stable`.

### Verification

Verification is the same as with the `edge` case.

# Updating Fleet Desktop

## 1. Edge Release

### Setup

The Verifier will setup a CentOS, Ubuntu, Windows and macOS host with `desktop` that uses the `edge` channel:
```sh
fleetctl package --type=pkg --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --desktop-channel edge
fleetctl package --type=msi --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --desktop-channel edge
fleetctl package --type=deb --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --desktop-channel edge
fleetctl package --type=rpm --fleet-url=https://fleet.example.com --enroll-secret=<...> --fleet-desktop --desktop-channel edge
```

> NOTE: The fleetctl version to use should be the latest released version, installed via `sudo npm install -g fleetctl`.

### Steps

Assuming version `vX.Y.Z` is being released.

1. Checkout a branch and edit the `FLEET_DESKTOP_VERSION` env variable in `.github/workflows/generate-desktop-targets.yml`.
2. Push and create a pull request.
3. Once the pull request is created a Github Action will be triggered
   [generate-desktop-targets.yml](https://github.com/fleetdm/fleet/actions/workflows/generate-desktop-targets.yml).
   It generates the desktop targets for macOS, Windows and Linux as artifacts.
4. Download the artifacts from the previous step and push them to the `edge` channel:
```sh
# Having extracted the asset for Linux in `./desktop.tar.gz`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target desktop \
    --platform linux \
    --name ./desktop.tar.gz \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Linux in `./desktop.app.tar.gz`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target desktop \
    --platform macos \
    --name ./desktop.app.tar.gz \
    --version X.Y.Z -t X.Y -t X -t edge

# Having extracted the asset for Windows in `./fleet-desktop.exe`
fleetctl updates add \
    --path $STAGING_TUF_PATH_LOCATION \
    --target desktop \
    --platform windows \
    --name ./fleet-desktop.exe \
    --version X.Y.Z -t X.Y -t X -t edge
```

### Verification

Verifier will make sure all the hosts have updated the target successfully. The update interval delay can be up to 15 minutes.

Currently, there's no direct way to verify the auto-update for the Fleet Desktop application.
One way to verify is to check for `INF exiting due to successful update` in the Orbit logs.

Once the Fleet Desktop Application has auto-updated on all hosts, Verifier runs the usual smoke testing on the 4 OSs (e.g. refetching & live querying hosts, listing software, etc.).

## 2. Stable Release

### Setup

Verifier runs the same setup as `edge`, but without setting the `--desktop-channel` flag (the default value is `stable`).

### Steps

Run the same `fleetctl updates add` command as the `edge` case with the same targets, but with `-t stable`.

### Verification

Verification is the same as with the `edge` case.
