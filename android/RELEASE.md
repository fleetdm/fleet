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

## 5. Test the RC

- Build and test the release candidate
- Fix any issues on the RC branch (cherry-pick fixes from main if applicable)

## 6. Build signed release

Ensure `keystore.properties` is configured with the release signing key.

```bash
./gradlew clean bundleRelease
```

Output: `app/build/outputs/bundle/release/app-release.aab`

## 7. Upload to Google Play

1. Go to [Google Play Console](https://play.google.com/console)
2. Select the Fleet app
3. Navigate to Release > Production (or appropriate track)
4. Upload the signed AAB
5. Add release notes
6. Review and roll out

## 8. Tag the release

After the release is uploaded, tag the RC branch:

```bash
git checkout rc-minor-android-v1.X.X
git tag fleetd-android-v1.X.X
git push origin rc-minor-fleetd-android-v1.X.X
```

## 9. Bring version bump and CHANGELOG to main

```bash
git checkout main
git checkout rc-minor-fleetd-android-v1.X.X -- app/build.gradle.kts CHANGELOG.md
git commit -m "Update version and CHANGELOG for fleetd-android-v1.X.X"
git push origin main
```
## 10. Update the info page

After the release is uploaded, update orbit/ANDROID_APP.md



This brings only the version bump and CHANGELOG updates to main, not other RC changes.
