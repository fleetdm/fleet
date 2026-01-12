# Fleet Android agent

- [Requirements](#requirements)
- [Building the project](#building-the-project)
- [Deploying via Android MDM](#deploying-via-android-mdm-development)
- [How the app starts](#how-the-app-starts)
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
keytool -genkeypair \
  -alias fleet-android \
  -keyalg RSA \
  -keysize 4096 \
  -validity 10000 \
  -keystore keystore.jks \
  -storepass YOUR_PASSWORD \
  -dname "CN=Your Name, O=Your Org, L=City, ST=State, C=US"
```

2. **Create `keystore.properties` file in the `android/` directory:**

```properties
storeFile=/path/to/keystore.jks
storePassword=YOUR_PASSWORD
keyAlias=fleet-android
keyPassword=YOUR_PASSWORD
```

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


### Getting the SHA256 fingerprint

The SHA256 fingerprint is required for MDM deployment. You can get it from your keystore.

```bash
keytool -list -v -keystore keystore.jks -alias fleet-android
# Grab SHA256 (remove colons and convert to base64)
echo <SHA256> | tr -d ':' | xxd -r -p | base64
```

Copy the fingerprint for use in the `mdm.android_agent.signing_sha256` config option.

## Deploying via Android MDM (development)

This feature requires setting the Android agent package config. Requires `FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS` to be set in your workarea to get the Google Play URL.

1. **Configure your Fleet server with the Android agent settings:**

Using environment variables:
```bash
export FLEET_MDM_ANDROID_AGENT_PACKAGE=com.fleetdm.agent.private.<yourname>
export FLEET_MDM_ANDROID_AGENT_SIGNING_SHA256=<SHA256 fingerprint>
```

Or in your Fleet config file:
```yaml
mdm:
  android_agent:
    package: com.fleetdm.agent.private.<yourname>
    signing_sha256: <SHA256 fingerprint>
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

## How the app starts

The Fleet Android agent is designed to run automatically without user interaction. The app starts in three scenarios:

### 1. On installation (COMPANION_APP role)

When the app is installed via MDM, Android Device Policy assigns it the `COMPANION_APP` role. This triggers `RoleNotificationReceiverService`, which starts the app process and runs `AgentApplication.onCreate()`.

### 2. On device boot

When the device boots, `BootReceiver` receives the `ACTION_BOOT_COMPLETED` broadcast and starts the app process, triggering `AgentApplication.onCreate()`.

### 3. Periodically every 15 minutes

`AgentApplication.onCreate()` schedules a `ConfigCheckWorker` to run every 15 minutes using WorkManager. This ensures the app wakes up periodically even if the process is killed.

**Note:** WorkManager ensures reliable background execution. The work persists across device reboots and process death.

### Why not ACTION_APPLICATION_RESTRICTIONS_CHANGED?

We don't use `ACTION_APPLICATION_RESTRICTIONS_CHANGED` to detect MDM config changes because:

1. This broadcast can only be registered dynamically (not in the manifest)
2. On Android 14+, [context-registered broadcasts are queued when the app is in cached state](https://developer.android.com/about/versions/14/behavior-changes-all#pending-broadcasts-queued)

This means the broadcast won't wake the app immediately when configs change if the app is in the background. WorkManager polling every 15 minutes is the reliable solution for detecting config changes.

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

### Integration tests (with real SCEP server)

Integration tests are skipped by default. To run them:

```bash
./gradlew test -PrunIntegrationTests=true \
  -Pscep.url=https://your-scep-server.com/scep \
  -Pscep.challenge=your-challenge-password
```

#### Setting Up a Test SCEP Server

Integration tests require a real SCEP server. Options:

1. **Production-grade SCEP servers:**
   - Microsoft NDES (Network Device Enrollment Service)
   - OpenXPKI
   - Ejbca

2. **Lightweight test servers:**
   - micromdm/scep (Docker)
   - jscep test server

### Docker SCEP Server (Easiest)

```bash
docker run -p 8080:8080 \
  -e SCEP_CHALLENGE=test-challenge-123 \
  micromdm/scep:latest
```

### Running Integration Tests

```bash
./gradlew test -PrunIntegrationTests=true \
  -Pscep.url=http://localhost:8080/scep \
  -Pscep.challenge=test-challenge-123
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
