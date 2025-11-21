# Fleet Android agent

- [Requirements](#requirements)
- [Building the project](#building-the-project)
- [Deploying via Android MDM](#deploying-via-android-mdm-development)
- [Running tests](#running-tests)
- [Code quality](#code-quality)
- [Troubleshooting](#troubleshooting)

## Requirements

- **JDK 17 or later** - Set `JAVA_HOME` environment variable
- **Android SDK** - Gradle finds it via:
  - `local.properties` file with `sdk.dir` (auto-created by Android Studio) ✅ Recommended
  - OR `ANDROID_HOME` / `ANDROID_SDK_ROOT` environment variables
  - Install via [Android Studio](https://developer.android.com/studio) (easiest)
  - Or install [command-line tools](https://developer.android.com/studio#command-line-tools-only)
  - Requires SDK Platform API 33+ and Build Tools 34.0.0+

## Building the project

### Debug build

```bash
./gradlew assembleDebug
```

Output: `app/build/outputs/apk/debug/app-debug.apk`

### Release build

```bash
./gradlew assembleRelease
```

Output: `app/build/outputs/apk/release/app-release.apk`

**Note:** By default (without signing configuration), this creates an **unsigned** APK not suitable for distribution.

### Signing release builds

Signing configuration is already set up in `build.gradle.kts`. You just need to provide the keystore and credentials.

**One-time setup per developer/machine:**

1. **Create a keystore:**

```bash
keytool -genkey -v -keystore keystore.jks \
  -alias fleet-android \
  -keyalg RSA -keysize 2048 -validity 10000
```

You'll be prompted for:
- **Password** (enter twice for confirmation) - This will be used for both keystore and key
- Your name, organization, location, etc.

2. **Create `keystore.properties` file in the `android/` directory:**

```properties
storeFile=path/to/keystore.jks
storePassword=your-password
keyAlias=fleet-android
keyPassword=your-password
```

**Note:** Use the same password you entered during keystore creation for both `storePassword` and `keyPassword`.

3. **Build signed release:**

```bash
# APK (for direct distribution)
./gradlew assembleRelease

# AAB (for Google Play Store)
./gradlew bundleRelease
```

Output:
- APK: `app/build/outputs/apk/release/app-release.apk`
- AAB: `app/build/outputs/bundle/release/app-release.aab`

**Verify signing:**

```bash
# APK - use apksigner (in SDK build-tools)
# Find your SDK and build-tools version:
grep sdk.dir local.properties
ls "$(grep sdk.dir local.properties | cut -d= -f2)/build-tools/"
# Then verify:
<sdk-path>/build-tools/<version>/apksigner verify --verbose app/build/outputs/apk/release/app-release.apk

# AAB - use jarsigner (included with JDK)
jarsigner -verify app/build/outputs/bundle/release/app-release.aab
```

## Deploying via Android MDM (development)

This feature is behind the feature flag `FLEET_DEV_ANDROID_AGENT_PACKAGE`. Requires `FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS` to be set in your workarea.

1. **Set the feature flag on your Fleet server:**

```bash
export FLEET_DEV_ANDROID_AGENT_PACKAGE=com.fleetdm.agent.private.<yourname>
```

2. **Change the `applicationId` in `app/build.gradle.kts`:**

```kotlin
defaultConfig {
    applicationId = "com.fleetdm.agent.private.<yourname>"
    // ...
}
```

3. **Build a signed release** (AAB) using the instructions above.

4. **Get the Google Play URL:**

```bash
# Run from top-level directory of the working tree
go run tools/android/android.go --command enterprises.webTokens.create --enterprise_id '<your-enterprise-id>'
```

5. **Upload your signed app** in the Private apps tab using the URL from the previous step.

6. **Wait ~10 minutes** for Google Play to process the upload.

7. **Enroll your Android device.**

The agent should start installing shortly. Check Google Play in your Work profile. If it shows as pending, try restarting the device.

### Full build with tests

```bash
./gradlew build
```

This runs:
- Compilation (debug + release)
- Unit tests
- Android Lint
- Spotless formatting checks (automatic)

## Running tests

### Unit tests (JVM)

```bash
./gradlew test
```

### Instrumented tests (requires emulator/device)

```bash
./gradlew connectedDebugAndroidTest
```

## Code quality

### Formatting with Spotless (ktlint)

**Check formatting:**
```bash
./gradlew spotlessCheck
```

**Auto-fix formatting issues:**
```bash
./gradlew spotlessApply
```

**Note:** Spotless checks run automatically during `./gradlew build`. Run `spotlessApply` to fix issues before committing.

### Static analysis with Detekt

**Run manually:**
```bash
./gradlew detekt
```

**Note:** Detekt does NOT run automatically in local builds (only in CI). Run manually when needed.

## Dependencies

See `gradle/libs.versions.toml` for complete list.

## Development workflow

1. **Before committing:** Run `./gradlew spotlessApply` to fix formatting
2. **Local verification:** Run `./gradlew build` to ensure everything passes
3. **Optional:** Run `./gradlew detekt` for static analysis
4. **Push:** CI will run all checks automatically

## Troubleshooting

**Clean build:**
```bash
./gradlew clean build
```

**Delete device from Android MDM:**
- Delete Work profile on Android device
- Using `tools/android/android.go`, delete the device and delete the associated policy (as of 2025/11/21, Fleet server does not do this)
