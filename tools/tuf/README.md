# Releasing updates to Fleet's TUF repository

The `releaser.sh` script automates the building and releasing of fleetd and osquery updates on [Fleet's TUF repository](https://tuf.fleetctl.com).

> - The script was developed and tested on macOS Intel.
> - It currently supports pushing new `fleetd` and `osqueryd` versions.
> - By storing credentials encrypted in a USB flash drive and storing their decryption passphrase on 1Password we are enforcing a form of 2FA.

```mermaid
graph LR;
    subgraph Workstation;
        releaser[releaser.sh];
        1password("<div><img src='../../website/assets/images/articles/tales-from-fleet-security-securing-1password-cover-1600x900@2x.jpg' /></div>1Password");
        usb("<div><img src='../../website/assets/images/articles/config-less-fleetd-agent-deployment-1600x900@2x.png' /></div>USB flash drive");
        repository[(./repository)];
    end;
    s3("<div><img src='../../website/assets/images/icon-aws-60x36@2x.png' /></div>s3://fleet-tuf-repo");
    github("<div><img src='../../website/assets/images/github-mark-white-24x24@2x.png' /></div>Github Action\n(signing and notarization)");
    
    usb--(1) copy encrypted signing keys-->releaser;
    1password--(2) get passphrases to decrypt encrypted signing keys-->releaser;
    1password--(3) get Github API token-->releaser;
    s3--(4) pull TUF repository-->releaser;
    releaser--(5) build components (new updates)\n(osqueryd, orbit, Fleet Desktop)-->github;
    github--(6) download built components-->releaser;
    releaser--(7) push updates and signed metadata-->s3;
```

## Permissions and configuration

Following is the checklist for all credentials and configuration needed to run the script.

### Dependencies

- `make`
- `git`
- 1Password 8 application.
- Install and configure 1Password's `op` cli to connect to the application: https://developer.1password.com/docs/cli/get-started/
- `aws` cli :`brew install awscli`.
- `fleetctl`: Either built from source or installed by npm.
- `tuf`: Download the release from https://github.com/theupdateframework/go-tuf/releases/download/v0.7.0/tuf_0.7.0_darwin_amd64.tar.gz and place the `tuf` executable in `/usr/local/bin/tuf`. You will need to make an exception in "Privacy & Security" because the executable is not signed.
- `gh`: `brew install gh`.

### 1Password

You need to create three passphrases on your private 1Password vault for encrypting the signing keys (more on signing keys below).
Create three private "passwords" with the following names: `TUF TARGETS`, `TUF SNAPSHOT` and `TUF TIMESTAMP`.
The resulting credentials will have the following "path" within 1Password (these paths will be provided to the `releaser.sh` script)
```sh
Private/TUF TARGETS/password
Private/TUF SNAPSHOT/password
Private/TUF TIMESTAMP/password
```

### AWS

The following is required to be able to run `aws` cli commands.

1. You will need to request the infrastructure team to add the "TUFAdministrators" role to your Google account.
2. Configure AWS SSO with the following steps: https://github.com/fleetdm/confidential/tree/main/infrastructure/sso#how-to-use-sso.
Set the profile name as `tuf` (the profile name will be provided to the `releaser.sh` script).
3. Test the access by running:
```sh
AWS_PROFILE=tuf aws sso login
```

### TUF signing keys

> You can skip this step if you already have authorized keys to sign and publish updates.

To release updates to our TUF repository you need the `root` role (ask in Slack who has such `root` role) to sign your signing keys.

1. First, run the following script
```sh
tuf gen-key targets && echo
tuf gen-key snapshot && echo
tuf gen-key timestamp && echo
```
2. Store the '$TUF_DIRECTORY/keys' folder (that contains the encrypted keys) on a USB flash drive that you will ONLY use for releasing fleetd updates.
3. Share '$TUF_DIRECTORY/staged/root.json' with Fleet member with the 'root' role, who will sign with its root key and push it to the remote repository.
4. The human with the `root` role will run the following commands to sign the provided `staged/root.json`:
```sh
tuf sign
tuf snapshot
tuf timestamp
tuf commit
```
And push the newly signed `root.json` to the remote repository.

### Encrypted keys in USB

For releasing fleetd you need to plug in the USB that contains encrypted signing keys.
In this guide we assume the USB device will be mounted in `/Volumes/FLEET-TUF/` and it ONLY contains a `keys/` directory.

### Github

#### Personal access token

> A personal access token is required to download artifacts from Github Actions using the Github API.

1. Create a fine-grained personal access token at https://github.com/settings/tokens?type=beta
2. Store the token on 1Password as a "password" with name "Github Token"
The resulting credential will have the following "path" within 1Password (this path will be provided to the script)
```sh
Private/Github Token/password
```

#### Github session

You need to log in to your Github account using the cli (`gh`).
```sh
gh auth login
```
It will be used to create a PR which is used to update the changelog and trigger the Github actions to build components.

## Samples

Following are samples of the script execution to release components to `edge` and `stable`.

> When releasing fleetd you need to checkout the branch (e.g. `main`) you want to release.

> NOTE: When releasing fleetd:
> If there are only `orbit` changes on a release we still have to release the `desktop` component with its version string bumped
> (even if there are no changes in it). This is due to the fact that we want users to see the new version in the tray icon,
> e.g. `"Fleet Desktop v1.21.0"`. Technical debt: We could improve this process to reduce the complexity of releasing
> fleetd when there are no Fleet Desktop changes.

### Releasing to `edge`

#### Releasing fleetd `1.23.0` to `edge`

```sh
AWS_PROFILE=tuf \
TUF_DIRECTORY=/Users/foobar/tuf.fleetctl.com \
COMPONENT=fleetd \
ACTION=release-to-edge \
VERSION=1.23.0 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-TUF/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/TUF TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/TUF SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/TUF TIMESTAMP/password" \
GITHUB_USERNAME=foobar \
GITHUB_TOKEN_1PASSWORD_PATH="Private/Github Token/password" \
PUSH_TO_REMOTE=1 \
./tools/tuf/releaser.sh
```

#### Releasing osquery `5.12.1` to `edge`

```sh
AWS_PROFILE=tuf \
TUF_DIRECTORY=/Users/foobar/tuf.fleetctl.com \
COMPONENT=osqueryd \
ACTION=release-to-edge \
VERSION=5.12.1 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-TUF/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/TUF TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/TUF SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/TUF TIMESTAMP/password" \
GITHUB_USERNAME=foobar \
GITHUB_TOKEN_1PASSWORD_PATH="Private/Github Token/password" \
PUSH_TO_REMOTE=1 \
./tools/tuf/releaser.sh
```

### Promoting from `edge` to `stable`

#### Promoting fleetd `1.23.0` from `edge` to `stable`

```sh
AWS_PROFILE=tuf \
TUF_DIRECTORY=/Users/foobar/tuf.fleetctl.com \
COMPONENT=fleetd \
ACTION=promote-edge-to-stable \
VERSION=1.23.0 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-TUF/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/TUF TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/TUF SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/TUF TIMESTAMP/password" \
PUSH_TO_REMOTE=1 \
./tools/tuf/releaser.sh
```

#### Promoting osqueryd `5.12.1` from `edge` to `stable`

```sh
AWS_PROFILE=tuf \
TUF_DIRECTORY=/Users/foobar/tuf.fleetctl.com \
COMPONENT=osqueryd \
ACTION=promote-edge-to-stable \
VERSION=5.12.1 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-TUF/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/TUF TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/TUF SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/TUF TIMESTAMP/password" \
PUSH_TO_REMOTE=1 \
./tools/tuf/releaser.sh
```

#### Releasing `swiftDialog` to `stable`

> `releaser.sh` doesn't support `swiftDialog` yet.
> macOS only component

The `swiftDialog` executable can be generated from a macOS host by running:
```sh
make swift-dialog-app-tar-gz version=2.2.1 build=4591 out-path=.
```
```sh
fleetctl updates add --target /path/to/macos/swiftDialog.app.tar.gz --platform macos --name swiftDialog --version 2.2.1 -t edge
```

#### Releasing `nudge` to `stable`

> `releaser.sh` doesn't support `nudge` yet.
> macOS only component

The `nudge` executable can be generated from a macOS host by running:
```sh
make nudge-app-tar-gz version=1.1.10.81462 out-path=.
```
```sh
fleetctl updates add --target /path/to/macos/nudge.app.tar.gz --platform macos --name nudge --version 1.1.10.81462 -t edge
```

#### Releasing `Escrow Buddy` to `stable`

> `releaser.sh` doesn't support `Escrow Buddy` yet.
> macOS only component

The `Escrow Buddy` pkg installer can be generated by running:
```sh
make escrow-buddy-pkg version=1.0.0 out-path=.
```
```sh
fleetctl updates add --target /path/to/escrowBuddy.pkg --platform macos --name escrowBuddy --version 1.0.0 -t stable
```

#### Updating timestamp

```sh
AWS_PROFILE=tuf \
TUF_DIRECTORY=/Users/foobar/tuf.fleetctl.com \
ACTION=update-timestamp \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-TUF/keys \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/TUF TIMESTAMP/password" \
PUSH_TO_REMOTE=1 \
./tools/tuf/releaser.sh
```

## Testing and improving the script

- You can specify `GIT_REPOSITORY_DIRECTORY` to set a separate path for the Fleet repository (it uses the current by default).
This is sometimes necessary if the tooling the script uses is not present in the branch we are trying to release from.
```sh
git clone git@github.com:fleetdm/fleet.git <SOME_DIRECTORY>
GIT_REPOSITORY_DIRECTORY=<SOME_DIRECTORY>
```

- If the PR and orbit tag were already generated but you need to run the script again you can set `SKIP_PR_AND_TAG_PUSH=1` to skip that part.

- While developing you can run with `PUSH_TO_REMOTE=0` to prevent pushing invalid metadata/components to the production repository.

## TODOs to improve releaser.sh

- Support releasing `nudge` and `swiftDialog`. 

## Troubleshooting

### Removing Unused Targets

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

### Invalid timestamp.json version

The following issue was solved by resigning the timestamp metadata `fleetctl updates timestamp` (executed three times to increase the version to `4175`)
```sh
2022-08-23T13:44:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
2022-08-23T13:59:48-03:00 INF update failed error="update metadata: update metadata: tuf: failed to decode timestamp.json: version 4172 is lower than current version 4174"
```
