# Release process

## 1. Create RC branch

```bash
git checkout main
git pull origin main
git checkout -b rc-minor-fleetd-android-v1.X.X
```

## 2. Update version numbers

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

## 3. Update CHANGELOG.md

From the repo root, run the changelog generator to pull entries from `android/changes/`:

```bash
make changelog-android version=1.X.X
```

This collects all entries from `android/changes/`, prepends them to `android/CHANGELOG.md` with a dated header, and stages the change files for deletion.

Review the generated changelog and manually add any additional entries that are not covered by the `android/changes/` directory.

## 4. Commit and push RC branch

```bash
git add android/app/build.gradle.kts android/CHANGELOG.md android/changes/
git commit -m "Prepare release v1.X.X"
git push origin rc-minor-fleetd-android-v1.X.X
```

## 5. Test the RC by releasing to the staging environment (com.fleetdm.agent.stage)

Prerequisites:
- Fleet server running with:
  - `export FLEET_MDM_ANDROID_AGENT_PACKAGE=com.fleetdm.agent.stage`
  - `export FLEET_MDM_ANDROID_AGENT_SIGNING_SHA256=uxe8ynMUe36j7avGtA2F4wHeA+gnQn6UbPP+7D3AbQQ=`
- In [Google Play Console](https://play.google.com/console) (using the "Google Play Admin" 1pass creds), add your Android MDM org ID to "Test and Release" --> "Advanced Settings" --> "Managed Google Play".
- Get the staging signing key from a previous releaser

### Build signed release

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

### Upload to Google Play

1. Go to [Google Play Console](https://play.google.com/console).
2. Select the Fleet staging app (`com.fleetdm.agent.stage`).
3. Navigate to "Test and release" > Production.
4. Select "Create new release"
5. Upload the signed .aab file.
6. Add release details at the bottom of the page.
7. After the .aab file has been processed, select Next, then Save, then select **Go to overview** in the modal that pops up.
8. You'll be redirected to **Publishing overview** page, where you need to select **Send 1 change for review**.
9. After Google approves the app, they will send an email to the Google Play console account.

### Test the release

Run through the testplans.

## 6. Release to production

Note: Only specific individuals have access to the release flow.

### Build signed release

Ensure `keystore.properties` is configured with the release signing key/password.

```bash
./gradlew clean bundleRelease
```

Output: `app/build/outputs/bundle/release/app-release.aab`

### Upload to Google Play

1. Go to [Google Play Console](https://play.google.com/console).
2. Select the Fleet app (`com.fleetdm.agent`).
3. Navigate to Release > Production.
4. Upload the signed .aab file.
5. Add release notes at the bottom of the page.
6. Select save, then select **Go to overview** in the modal that pops up.
7. You'll be redirected to **Publishing overview** page, where you need to select **Sent to review**.
8. After Google approves the app, they will send an email to the main Google Play console account.

## 7. Tag the release

After the release is uploaded, tag the RC branch:

```bash
git checkout rc-minor-fleetd-android-v1.X.X
git tag fleetd-android-v1.X.X
git push origin rc-minor-fleetd-android-v1.X.X
```

## 8. Bring version bump and CHANGELOG to main

```bash
git checkout main
git pull origin main
git checkout -b bring-fleetd-android-v1.X.X-to-main
git checkout rc-minor-fleetd-android-v1.X.X -- android/app/build.gradle.kts android/CHANGELOG.md
git diff --name-only --diff-filter=D main...rc-minor-fleetd-android-v1.X.X -- android/changes/ | xargs git rm --ignore-unmatch
git commit -m "Update version and CHANGELOG for fleetd-android-v1.X.X"
git push origin bring-fleetd-android-v1.X.X-to-main
```
## 10. Update the info page

After the release is uploaded, update orbit/ANDROID_APP.md with the new release version.



Then open a PR to merge `bring-fleetd-android-v1.X.X-to-main` into `main`.

This brings the version bump and CHANGELOG updates to main and removes only the changelog entries that were processed in the RC, preserving any new entries added to main after the RC branch was cut.
