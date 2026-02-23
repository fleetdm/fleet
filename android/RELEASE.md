# Release process

## 1. Create RC branch

```bash
git checkout main
git pull origin main
git checkout -b rc-minor-fleetd-android-v1.X.X
```

## 2. Update version numbers

In `app/build.gradle.kts`, update:

```kotlin
defaultConfig {
    applicationId = "com.fleetdm.agent.stage"
    versionCode = 2          // Increment by 1 each release
    versionName = "1.1.0"    // Semantic version for display
}
```

- `versionCode`: Integer that must increase with each release (Google Play requirement)
- `versionName`: Human-readable version string shown to users

## 3. Update CHANGELOG.md

Add an entry for the new version with changes since the last release.

## 4. Commit and push RC branch

```bash
git add app/build.gradle.kts CHANGELOG.md
git commit -m "Prepare release v1.1.0"
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
7. Select Next, then Save, then select **Go to overview** in the modal that pops up.
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
git commit -m "Update version and CHANGELOG for fleetd-android-v1.X.X"
git push origin bring-fleetd-android-v1.X.X-to-main
```

Then open a PR to merge `bring-fleetd-android-v1.X.X-to-main` into `main`.

This brings only the version bump and CHANGELOG updates to main, not other RC changes.
