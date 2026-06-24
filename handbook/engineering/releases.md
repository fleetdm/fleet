# Releases

This handbook page details Fleet's release process, including QA, release candidate management, agent releases, and post-release tasks.


## Participate in QA Day

Once per sprint, each product group is expected to take a day to assist in QA-related activities. On that day, generally the most straightforward way to assist the QA team is to validate issues in the `Awaiting QA` stage marked with the `~assisting-qa` label. Start with issues milestoned for the lowest-version-number active release candidate, and clear your product group's queue for that release before assisting another team with QA. You may not QA issues where you made code changes, to ensure that two people run through the test plan (the implementing engineer and the person performing QA).

For each issue:

1. Add yourself as an assignee when you start QA. If other work comes up that prevents you from completing the QA process, remove yourself as an assignee to ensure someone else picks the issue up.
2. Validate the changes, either via the test plan (for stories) or by reproducing the bug on an older version and the fix in the current version (for bugs).
3. Document QA steps performed and outcome in a comment on the story (not subtask) or bug.
4. If changes are needed to make QA pass, either create an unreleased bug (if changes required are small relative to the size of the original bug or story, e.g. a missed edge case) or move the issue (and relevant subtasks, if there are any) back to `In progress` (if changes required are significant relative to the size of the ticket, e.g. if an item listed in the test plan fails). Mention in the relevant product group's Slack channel when you take either of these actions to ensure QA failures are addressed quickly (e.g. the product group's tech lead may need to assign an unreleased bug fix to an engineer other than the developer(s) on the original bug or story).
5. Once QA passes, move the issue to `Ready for release`.


## Run Fleet locally for QA purposes

To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name. For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`.

To start a preview without starting the simulated hosts, use the `--no-hosts` flag (e.g., `fleetctl preview --no-hosts`).

For each bug found, please use the [bug report template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) to create a new bug report issue.

For unreleased bugs in an active sprint, a new bug is created with the `~unreleased bug` label. The `:release` label and associated product group label is added, and the milestone is set to the version that the feature will be released in. For example, if the feature will be released in v4.71.0 and the bug did not exist prior to that version, the milestone is set to `v4.71.0`. The engineer responsible for the feature is assigned. If QA is unsure who the bug should be assigned to, it is assigned to the EM. Fixing the bug becomes part of the story.


## Fleet server releases

### Create a release candidate

All minor releases go through the release candidate process before they are published. A release candidate for the next minor release is created on the first Monday of the next sprint at 8:00 AM Pacific (see [Fleet's release calendar](https://calendar.google.com/calendar/u/0?cid=Y192Nzk0M2RlcW4xdW5zNDg4YTY1djJkOTRic0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t)). A release candidate branch is created at `rc-minor-fleet-v4.x.x` and no additional feature work or released bug fixes are merged without EM and QA approval.

1. [Run the first step](https://github.com/fleetdm/fleet/tree/main/tools/release#minor-release-typically-end-of-sprint) of the minor release section of the Fleet releases script to create the release candidate branch, the release QA issue, and announce the release candidate in Slack.

2. Open the [confidential repo environment variables](https://github.com/fleetdm/confidential/settings/variables/actions) page and update the `QAWOLF_DEPLOY_TAG` repository variable with the name of the release candidate branch.

During the release candidate period, the release candidate is deployed to our QA Wolf instance every morning instead of `main` to ensure that any new bugs reported by QA Wolf are in the upcoming release and need to be fixed before publishing the release.

### Merge unreleased bug fixes into the release candidate

Only merge unreleased bug fixes during the release candidate period to minimize code churn and help ensure a stable release. To merge a bug fix into the release candidate:

1. Merge the fix into `main`. 
2. `git checkout` the release candidate branch and create a new local branch. 
3. `git cherry-pick` your commit from `main` into your new local branch.
4. Create a pull request from your new branch to the release candidate. 

This process ensures your bug fix is included in `main` for future releases, as well as the release candidate branch for the pending release.

If there is partially merged feature work when the release candidate is created, the previously merged code must be reverted. If there is an exceptional, business-critical need to merge feature work into the release candidate, as determined by the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals), the release candidate [feature merge exception process](#request-release-candidate-feature-merge-exception) may be followed.

### Request release candidate feature merge exception

1. Notify product group EM that feature work will not merge into `main` before the release candidate is cut and requires a feature merge exception.
2. EM notifies QA lead for the product group and the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals).
3. EM, QA lead, and [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) must all approve the feature work PR before it is merged into the release candidate branch.

> This exception process should be avoided whenever possible. Any feature work merged into the release candidate will likely result in a significant release delay.

### Prepare Fleet release

See the ["Releasing Fleet" contributor guide](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/releasing-fleet.md).

### Deploy a new release to dogfood

After each Fleet release, the new release is deployed to Fleet's "dogfood" (internal) instance. Because dogfood is used for live sales demos, coordinate deploy timing as follows:

- **During U.S. business hours (7am - 5pm Pacific, Monday - Friday):** `@mention` Allen Houchins (`@allenhouchins`) in Slack before deploying so he can confirm no demos are scheduled. Allen owns deploy approval during business hours.
- **Outside U.S. business hours:** deploys are okay without coordination. We accept the risk of occasionally interrupting an international demo in exchange for unblocking releases.

How to deploy a new release to dogfood:

1. Head to the **Tags** page on the fleetdm/fleet Docker Hub: https://hub.docker.com/r/fleetdm/fleet/tags
2. In the **Filter tags** search bar, type in the latest release (ex. v4.19.0).
3. Locate the tag for the new release and copy the image name. An example image name is "fleetdm/fleet:v4.19.0".
4. Head to the "Deploy Dogfood Environment" action on GitHub: https://github.com/fleetdm/fleet/actions/workflows/dogfood-deploy.yml
5. Select **Run workflow** and paste the image name in the **The image tag wished to be deployed.** field.

> Note that this action will not handle down migrations. Always deploy a newer version than is currently deployed.
>
> Note that "fleetdm/fleet:main" is not an image name, instead use the commit hash in place of "main".


## Fleetd releases (macOS, Windows, Linux)

Fleetd for macOS, Windows and Linux is an agent composed of several components. The latest released versions in TUF are documented in the [TUF version tracking doc](https://github.com/fleetdm/fleet/blob/main/orbit/TUF.md).

### Compatibility rules

#### Why do we need a strategy?

Fleetd and Fleet use different release strategies. Fleetd components are updated via "automatic updates" by continuously polling https://tuf.fleetctl.com/ for new versions, whereas on-premises Fleet servers are updated manually by administrators.
For this reason we need a good release strategy to not break on-premise deployments when we release new versions of Fleetd components.

#### Must rule

"New Fleetd versions always support communication + operation with older Fleet servers."

> Why is it a must?

As mentioned before, Fleetd uses an auto-update mechanism, whereas Fleet does not.
We don't want to break on-premise Fleet deployments, and we don't want to force Fleetd users to update their servers every time we push a new Fleetd update to Fleet's TUF server.

#### Nice to have

Nice to have, but not a must: "New Fleet server versions support old versions of Fleetd."

> Why is it not a must?

This allows some flexibility when developing new features in Fleetd and Fleet.

#### Release process

1. Fleetd components (Orbit, Fleet Desktop and osqueryd) must be released to FleetDM's TUF before new Fleet server releases are available in Github.
2. When the new Fleet server version doesn't support older Fleetd versions (see [Nice to have](#nice-to-have)), the release notes must document their minimum supported Fleetd version. This is for users that use Fleetd with auto-updates disabled or they pin to a specific channel. These users would need to first update Fleetd on their devices and then proceed to upgrade Fleet server.

### Prepare and publish fleetd release

For one-time setup of credentials and tools needed to run the release script, see the [releaser.sh setup guide](https://github.com/fleetdm/fleet/blob/main/tools/tuf/README.md#permissions-and-configuration).

> When releasing fleetd you need to checkout the branch (e.g. `main`) you want to release.

> NOTE: When releasing fleetd:
> If there are only `orbit` changes on a release we still have to release the `desktop` component with its version string bumped
> (even if there are no changes in it). This is due to the fact that we want users to see the new version in the tray icon,
> e.g. `"Fleet Desktop v1.21.0"`. Technical debt: We could improve this process to reduce the complexity of releasing
> fleetd when there are no Fleet Desktop changes.

#### Releasing to `edge`

##### Releasing fleetd `1.23.0` to `edge`

1. Push to staging:
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
COMPONENT=fleetd \
ACTION=release-to-edge \
VERSION=1.23.0 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
GITHUB_USERNAME=foobar \
GITHUB_TOKEN_1PASSWORD_PATH="Private/Github Token/password" \
./tools/tuf/releaser.sh
```
2. Smoke test release on staging.
3. Push to production:
```sh
ACTION=release-to-production \
COMPONENT=fleetd \
VERSION=1.23.0 \
./tools/tuf/releaser.sh
```
4. Create PR with changelog against `main` (it may fail with conflicts when using patch branches):
```sh
ACTION=create-fleetd-release-pr \
VERSION=1.23.0 \
./tools/tuf/releaser.sh
```
4. Smoke test release on production.


##### Releasing osquery `5.12.1` to `edge`

1. Push to staging:
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
COMPONENT=osqueryd \
ACTION=release-to-edge \
VERSION=5.12.1 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
GITHUB_USERNAME=foobar \
GITHUB_TOKEN_1PASSWORD_PATH="Private/Github Token/password" \
./tools/tuf/releaser.sh
```
2. Smoke test release on staging.
3. Push to production:
```sh
ACTION=release-to-production ./tools/tuf/releaser.sh
```
4. Smoke test release on production.

#### Promoting from `edge` to `stable`

##### Promoting fleetd `1.23.0` from `edge` to `stable`

1. Push to staging:
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
COMPONENT=fleetd \
ACTION=promote-edge-to-stable \
VERSION=1.23.0 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
./tools/tuf/releaser.sh
```
2. Smoke test release on staging.
3. Push to production:
```sh
ACTION=release-to-production ./tools/tuf/releaser.sh
```
4. Smoke test release on production.

##### Promoting osqueryd `5.12.1` from `edge` to `stable`

1. Push to staging:
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
COMPONENT=osqueryd \
ACTION=promote-edge-to-stable \
VERSION=5.12.1 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
./tools/tuf/releaser.sh
```
2. Smoke test release on staging.
3. Push to production:
```sh
ACTION=release-to-production ./tools/tuf/releaser.sh
```
4. Smoke test release on production.
5. Update osquery schema and flags:
```sh
ACTION=update-osquery-schema \
VERSION=5.19.0 \
./tools/tuf/releaser.sh
```

#### Releasing `swiftDialog` to `stable`

> macOS only component

1. Download `swiftDialog` from the desired run of https://github.com/fleetdm/fleet/actions/workflows/generate-swift-dialog-targets.yml.
2. Extract the downloaded `.zip` to `/path/to/swiftDialog.app.tar.gz`
3. Push to staging:  
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
ACTION=release-swiftDialog-to-stable \
VERSION=2.5.6 \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
SWIFT_DIALOG_PATH=/path/to/swiftDialog.app.tar.gz \
./tools/tuf/releaser.sh
```
4. Push to production:
```sh
ACTION=release-to-production ./tools/tuf/releaser.sh
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

1. Push to staging:
```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
ACTION=update-timestamp \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
./tools/tuf/releaser.sh
```
2. Push to production:
```sh
ACTION=release-to-production ./tools/tuf/releaser.sh
```

#### Doing a patch release of fleetd

Patch releases follow the same process as releasing a minor version, except instead of checking out the `main` branch of Fleet locally, you check out a patch branch of Fleet, e.g.:

```
git checkout rc-minor-fleetd-v1.41.1
```

As always, the `VERSION` env var used when running `releaser.sh` should match the version of the fleetd release, e.g.

```sh
TUF_DIRECTORY=/Users/foobar/updates-staging.fleetdm.com \
COMPONENT=fleetd \
ACTION=release-to-edge \
VERSION=1.41.1 # <-- note the patch version \
KEYS_SOURCE_DIRECTORY=/Volumes/FLEET-UPD/keys \
TARGETS_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TARGETS/password" \
SNAPSHOT_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES SNAPSHOT/password" \
TIMESTAMP_PASSPHRASE_1PASSWORD_PATH="Private/UPDATES TIMESTAMP/password" \
GITHUB_USERNAME=foobar \
GITHUB_TOKEN_1PASSWORD_PATH="Private/Github Token/password" \
./tools/tuf/releaser.sh
```

See https://github.com/fleetdm/fleet/blob/main/orbit/TUF.md to find the latest released version.

After following the rest of the "Releasing to edge" steps above, publish your release using the instructions in "Promoting from edge to stable" above as you would for a minor release, again remembering to set the `VERSION` accordingly.


## Fleetd Android releases

### 1. Create RC branch

```bash
git checkout main
git pull origin main
git checkout -b rc-minor-fleetd-android-v1.X.X
```

### 2. Update version numbers

In `app/build.gradle.kts`, update versionCode and versionName:

```kotlin
defaultConfig {
    applicationId = "com.fleetdm.agent"
    versionCode = 2          // Increment by 1 each release
    versionName = "1.1.0"    // Semantic version for display
}
```

- `versionCode`: Integer that must increase with each release (Google Play requirement)
- `versionName`: Human-readable version string shown to users

### 3. Update CHANGELOG.md

From the repo root, run the changelog generator to pull entries from `android/changes/`:

```bash
make changelog-android version=1.X.X
```

This collects all entries from `android/changes/`, prepends them to `android/CHANGELOG.md` with a dated header, and stages the change files for deletion.

Review the generated changelog and manually add any additional entries that are not covered by the `android/changes/` directory.

### 4. Commit and push RC branch

```bash
git add android/app/build.gradle.kts android/CHANGELOG.md android/changes/
git commit -m "Prepare release v1.X.X"
git push origin rc-minor-fleetd-android-v1.X.X
```

### 5. Test the RC by releasing to the staging environment (com.fleetdm.agent.stage)

Prerequisites:
- Fleet server running with:
  - `export FLEET_MDM_ANDROID_AGENT_PACKAGE=com.fleetdm.agent.stage`
  - `export FLEET_MDM_ANDROID_AGENT_SIGNING_SHA256=uxe8ynMUe36j7avGtA2F4wHeA+gnQn6UbPP+7D3AbQQ=`
- In [Google Play Console](https://play.google.com/console) (using the "Google Play Admin" 1pass creds), add your Android MDM org ID to "Test and Release" --> "Advanced Settings" --> "Managed Google Play".
- Get the staging signing key from a previous releaser

#### Build signed release

In `app/build.gradle.kts`, update the application ID:

```kotlin
defaultConfig {
    applicationId = "com.fleetdm.agent.stage"
}
```

Ensure `keystore.properties` is configured with the staging signing key:

```
storeFile=./qa-keystore.jks
storePassword=<get-this-from-a-previous-releaser>
keyAlias=fleet-android
keyPassword=<get-this-from-a-previous-releaser>
```

```bash
./gradlew clean bundleRelease
```

Output: `app/build/outputs/bundle/release/app-release.aab`

#### Upload to Google Play

1. Go to [Google Play Console](https://play.google.com/console).
2. Select the Fleet staging app (`com.fleetdm.agent.stage`).
3. Navigate to "Test and release" > Production.
4. Select "Create new release"
5. Upload the signed .aab file.
6. Add release details at the bottom of the page.
7. After the .aab file has been processed, select Next, then Save, then select **Go to overview** in the modal that pops up.
8. You'll be redirected to **Publishing overview** page, where you need to select **Send 1 change for review**.
9. After Google approves the app, they will send an email to the Google Play console account.

#### Test the release

Run through the testplans.

### 6. Release to production

Note: Only specific individuals have access to the release flow.

#### Build signed release

Ensure `keystore.properties` is configured with the release signing key/password.

```bash
./gradlew clean bundleRelease
```

Output: `app/build/outputs/bundle/release/app-release.aab`

#### Upload to Google Play

1. Go to [Google Play Console](https://play.google.com/console).
2. Select the Fleet app (`com.fleetdm.agent`).
3. Navigate to Release > Production.
4. Upload the signed .aab file.
5. Add release notes at the bottom of the page.
6. Select save, then select **Go to overview** in the modal that pops up.
7. You'll be redirected to **Publishing overview** page, where you need to select **Sent to review**.
8. After Google approves the app, they will send an email to the main Google Play console account.

### 7. Tag the release

After the release is uploaded, tag the RC branch:

```bash
git checkout rc-minor-fleetd-android-v1.X.X
git tag fleetd-android-v1.X.X
git push origin rc-minor-fleetd-android-v1.X.X
```

### 8. Bring version bump and CHANGELOG to main

```bash
git checkout main
git pull origin main
git checkout -b bring-fleetd-android-v1.X.X-to-main
git checkout rc-minor-fleetd-android-v1.X.X -- android/app/build.gradle.kts android/CHANGELOG.md
git diff --name-only --diff-filter=D main...rc-minor-fleetd-android-v1.X.X -- android/changes/ | xargs git rm --ignore-unmatch
git commit -m "Update version and CHANGELOG for fleetd-android-v1.X.X"
git push origin bring-fleetd-android-v1.X.X-to-main
```

Then open a PR to merge `bring-fleetd-android-v1.X.X-to-main` into `main`.

This brings the version bump and CHANGELOG updates to main and removes only the changelog entries that were processed in the RC, preserving any new entries added to main after the RC branch was cut.

### 9. Conclude the milestone

After the release is published, close out the associated GitHub issues and milestone. For the canonical checklist, see [Conclude current milestone](#conclude-current-milestone):

1. **Move associated stories to the Drafting board**: Stories may not be on the [Drafting](https://github.com/orgs/fleetdm/projects/67) board (project 67) yet. For each `story`-labeled issue in the `fleetd-android-v1.X.X` milestone, apply the `:product` label and remove the `:release` label so it moves onto the product drafting board.

2. **Move associated stories to "Confirm and celebrate"**: On the [Drafting](https://github.com/orgs/fleetdm/projects/67) board, filter by the `fleetd-android-v1.X.X` milestone, and move all stories to the "Confirm and celebrate" column. Product will close these issues during their [confirm and celebrate ritual](https://fleetdm.com/handbook/product#rituals). [Engineering-initiated stories](https://fleetdm.com/handbook/engineering#create-an-engineering-initiated-story) (`~engineering-initiated` label) can be closed without confirm and celebrate.

3. **Close associated bugs**: Close the remaining (non-story) issues in the `fleetd-android-v1.X.X` milestone.

4. **Close the GitHub milestone**: Visit [GitHub's milestone page](https://github.com/fleetdm/fleet/milestones) and close the `fleetd-android-v1.X.X` milestone.

5. **Announce**: Announce that the release milestone has been closed in #help-engineering.


## Fleetd ChromeOS releases

The Chrome extension is released via [Google Admin](https://admin.google.com).
For testing, use the [test extension deployment guide](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/deploying-chrome-test-ext.md).
For production releases, follow the [Chrome extension
README](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/README.md).


## Confirm latest versions of dependencies

Before kicking off release QA, confirm that we are using the latest versions of dependencies we want to keep up-to-date with each release. Currently, those dependencies are:

1. **Go**: Latest minor release
- Check the [Go version specified in Fleet's go.mod file](https://github.com/fleetdm/fleet/blob/main/go.mod) (`go 1.XX.YY`).
- Check the [latest minor version of Go](https://go.dev/dl/). For example, if we are using `go1.19.8`, and there is a new minor version `go1.19.9`, we will upgrade.
- If the latest minor version is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the current oncall engineer. Add the `~release blocker` label. We must upgrade to the latest minor version before publishing the next release.
- If the latest major version is greater than the version included in Fleet, [create a story](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story%2C%3Aproduct&projects=&template=story.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the current oncall engineer. This will be considered for an upcoming sprint. The release can proceed without upgrading the major version.

> In Go versioning, the number after the first dot is the "major" version, while the number after the second dot is the "minor" version. For example, in Go 1.19.9, "19" is the major version and "9" is the minor version. Major version upgrades are assessed separately by engineering.

Our goal is to keep these dependencies up-to-date with each release of Fleet. If a release is going out with an old dependency version, it should be treated as a [critical bug](https://fleetdm.com/handbook/engineering#critical-bugs) to make sure it is updated before the release is published.

3. **osquery**: Latest release
- Check the [latest version of osquery](https://github.com/osquery/osquery/releases).
- Check the [version included in Fleet](https://github.com/fleetdm/fleet/blob/main/.github/workflows/generate-osqueryd-targets.yml#L27).
- If the latest release of osquery is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#on-call-engineer).
- Do not add the `~release blocker` label. 
- Update the bug description to note that changes to [osquery command-line flags](https://osquery.readthedocs.io/en/stable/installation/cli-flags/) require updates to Fleet's flag validation and related documentation [as shown in this pull request](https://github.com/fleetdm/fleet/pull/16239/files). 

4. Vulnerability data sources
- Check the [NIST National Vulnerability Database website](https://nvd.nist.gov/) for any announcements that might impact our [NVD data feed](https://github.com/fleetdm/fleet/blob/5e22f1fb4647a6a387ca29db6dd75d492f1864d6/cmd/cpe/generate.go#L53). 
- Check the [CISA website](https://www.cisa.gov/) for any news or announcements that might impact our [CISA data feed](https://github.com/fleetdm/fleet/blob/5e22f1fb4647a6a387ca29db6dd75d492f1864d6/server/vulnerabilities/nvd/sync.go#L137). 

If an announcement is found for either data source that may impact data feed availability, notify the current [on-call engineer](https://fleetdm.com/handbook/engineering#on-call-engineer). Notify them that it is their responsibility to investigate and file a bug or take further action as necessary. 

5. Vulnerability OS coverage
- Check whether any new major operating system versions have been released since the last check.
- **Windows**: Verify that new Windows Server and Windows desktop versions are included in the [MSRC product mapping](https://github.com/fleetdm/fleet/blob/main/server/vulnerabilities/msrc/parsed/product.go).

If a new OS version is missing, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=).


## Indicate your product group is release-ready

Once a product group completes its QA process during the release candidate period, its QA lead moves the smoke testing ticket to the "Ready for release" column on their GitHub board. They then notify the release ritual DRI by tagging them in a comment, indicating that their group is prepared for release. The release ritual DRI starts the [release process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/releasing-fleet.md) after all QA leads have made these updates and confirmed their readiness for release.


## Submit test coverage requests to QA Wolf

Fleet QA owns the test planning process and identifies what needs to be automated. After each sprint, we review merged PRs, release notes, and demo recordings to find new automation candidates.
We track these in a shared [Google Doc](https://docs.google.com/document/d/1jr8wxZZNTvcAB2IMOrsqY4NTW4eceX-3CABiYKpb_pY/edit?usp=sharing) and categorize them as:
- New test requests (feature + what to test)
- Existing tests to update

Once coverage is agreed on, Fleet QA submits the request via [QA Wolf's Coverage Request form](https://app.qawolf.com/fleet/coverage-requests). The most recent sprints are prioritized first.
This workflow lets QA Wolf focus on test implementation while Fleet QA stays accountable for identifying clear, high-value test needs.


## Conclude current milestone 

Immediately after publishing a new release of Fleet or fleetd, close out the associated GitHub issues and milestones. 

1. **Update product group boards**: In GitHub Projects, go to each product group board tracking the current release and filter by the current milestone.

2. **Move user stories to drafting board**: Select all items in "Ready for release" that have the `story` label. Apply the `:product` label and remove the `:release` label. These items will move back to the product drafting board.

3. **Confirm and close**: Make sure that all items with the `story` label have left the "Ready for release" column. Select all remaining items in the "Ready for release" column and move them to the "Closed" column. This will close the related GitHub issues.

4. **Confirm and celebrate**: Open the [Drafting](https://github.com/orgs/fleetdm/projects/67) board. Filter by the current milestone and move all stories to the "Confirm and celebrate" column. Product will close the issues during their [confirm and celebrate ritual](https://fleetdm.com/handbook/product#rituals). [Engineering-initiated stories](https://fleetdm.com/handbook/engineering#create-an-engineering-initiated-story) (`~engineering-initiated` label) can be closed without confirm and celebrate.

5. **Close GitHub milestone**: Visit [GitHub's milestone page](https://github.com/fleetdm/fleet/milestones) and close the current milestone.

6. Announce that the release milestone has been closed in #help-engineering.

7. Visit the [confidential repo variables](https://github.com/fleetdm/confidential/settings/variables/actions) page and update the `QAWOLF_DEPLOY_TAG` repository variable to `main` so that the latest code is deployed to QA Wolf every morning.


## Update the Fleet releases calendar

The [Fleet releases Google calendar](https://calendar.google.com/calendar/embed?src=c_v7943deqn1uns488a65v2d94bs%40group.calendar.google.com&ctz=America%2FChicago) is kept up-to-date by the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals). Any change to targeted release dates is reflected on this calendar.

When target release dates are changed on the calendar, the release ritual DRI also updates the milestone due date.


## Discuss release dates

A single Slack thread is created in the #help-releases channel for every release candidate. Any discussions about release dates should be kept within the release candidate's thread.


## Handle process exceptions for non-released code

Some of our code does not go through a scheduled release process and is released immediately via GitHub workflows:

1. The [fleetdm/nvd](https://github.com/fleetdm/nvd) repository.
2. The [fleetdm/vulnerabilities](https://github.com/fleetdm/vulnerabilities) repository.
3. Our [website](https://github.com/fleetdm/fleet/tree/main/website) directory.

In these cases there are two differences in our pull request process:

- QA is done before merging the code change to the main branch.
- Tickets are not moved to "Ready for release". Bugs are closed, and user stories are moved to the product drafting board's "Confirm and celebrate" column.


<meta name="maintainedBy" value="lukeheath">
<meta name="description" value="Fleet's release process, including QA, release candidates, agent releases, and post-release tasks.">
