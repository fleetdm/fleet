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

## Osquery on Android

This project extends the Fleet Android agent toward an Android osquery runtime that executes distributed queries and reports results back to Fleet.  
The goal is to make Android hosts first-class query targets with osquery-style tables and behavior.

### Enrollment summary

The app reads `server_url`, `enroll_secret`, and `host_uuid` from Android managed configuration.  
It enrolls with Fleet Orbit and stores the node key securely on device for future API calls.  
In debug builds, if managed configuration is missing, it can fall back to debug-provided server URL and enroll secret.

### Query polling cadence

For distributed queries, the app checks in with Fleet on a loop using WorkManager.  
In the current implementation, the debug polling interval is hardcoded to **15 seconds** in `DistributedCheckinWorker`.  
This interval is **not configurable yet**; making it configurable (for example via managed config or a build setting) is a good next step.

### Android table reference

| Table | Summary | Quick query |
| --- | --- | --- |
| `installed_apps` | Installed app inventory and versions | `SELECT app_name, package_name, version_name FROM installed_apps LIMIT 25;` |
| `app_permissions` | App permissions and grant state | `SELECT app_name, permission, granted FROM app_permissions LIMIT 50;` |
| `os_version` | Android version/build/security patch info | `SELECT name, version, build, security_patch FROM os_version;` |
| `osquery_info` | Agent runtime metadata | `SELECT uuid, instance_id, version FROM osquery_info;` |
| `certificates` | Certificate records visible to the agent | `SELECT alias, subject, issuer, not_after FROM certificates LIMIT 50;` |
| `device_info` | Device model/manufacturer/hardware metadata | `SELECT manufacturer, brand, model, device FROM device_info;` |
| `network_interfaces` | Interface and addressing details | `SELECT name, mac, mtu, addresses FROM network_interfaces;` |
| `battery` | Battery state and health | `SELECT percent_remaining, charging, health FROM battery;` |
| `wifi_networks` | Wi-Fi connection/network details | `SELECT ssid, bssid, rssi, is_connected FROM wifi_networks;` |
| `system_properties` | Android system property key/value pairs | `SELECT key, value FROM system_properties LIMIT 50;` |
| `android_logcat` | Recent logcat entries | `SELECT timestamp, level, tag, message FROM android_logcat LIMIT 100;` |
| `time` | Current local time/timezone snapshot | `SELECT weekday, hour, minutes, local_timezone, unix_time FROM time;` |
| `uptime` | Device uptime duration snapshot | `SELECT days, hours, minutes, seconds, total_seconds FROM uptime;` |
| `system_info` | Host identity, hardware, and memory summary snapshot | `SELECT hostname, uuid, hardware_vendor, hardware_model, physical_memory FROM system_info;` |
| `kernel_info` | Kernel and runtime version snapshot | `SELECT version, release, build, platform FROM kernel_info;` |
| `memory_info` | Current memory totals and low-memory state | `SELECT total_bytes, available_bytes, threshold_bytes, low_memory FROM memory_info;` |
| `processes` | Visible process snapshot | `SELECT pid, name, uid, package_name, importance FROM processes LIMIT 50;` |
| `interface_addresses` | Interface-to-address rows | `SELECT interface, address, family FROM interface_addresses LIMIT 50;` |
| `routes` | Best-effort route snapshot | `SELECT destination, gateway, interface FROM routes LIMIT 50;` |
| `users` | Current app/profile identity row | `SELECT uid, gid, username, directory FROM users;` |
| `mounts` | Filesystem mount snapshot | `SELECT device, path, type, flags FROM mounts LIMIT 50;` |
| `cpu_info` | CPU capability snapshot | `SELECT cores, arch, model, hardware, vendor FROM cpu_info;` |

### Quick start (5 minutes)

Run these commands from `android/`:

```bash
adb devices
adb reverse tcp:8080 tcp:8080
export FLEET_ENROLL_SECRET='YOUR_ENROLL_SECRET'
FLEET_SERVER_URL='http://127.0.0.1:8080' ./gradlew installDebug
adb shell pm clear com.fleetdm.agent
adb shell monkey -p com.fleetdm.agent -c android.intent.category.LAUNCHER 1
```

### Verify it works

1. Open Fleet UI and check that the Android host appears in **Hosts**.
2. Run a simple live query:

```sql
SELECT name, version, platform, security_patch FROM os_version;
```

3. Confirm logs show successful read/write loop:

```bash
adb logcat | rg "fleet-ApiClient|fleet-distributed|Successfully enrolled host|distributed/read"
```

### QA quick checks for new cross-OS tables

Run these in Fleet live query and confirm exactly 1 row each:

```sql
SELECT * FROM time;
SELECT * FROM uptime;
SELECT * FROM system_info;
SELECT * FROM kernel_info;
SELECT * FROM memory_info;
SELECT * FROM processes LIMIT 20;
SELECT * FROM interface_addresses LIMIT 20;
SELECT * FROM routes LIMIT 20;
SELECT * FROM users;
SELECT * FROM mounts LIMIT 20;
SELECT * FROM cpu_info;
```

Expected sanity checks:
- `time.unix_time` is close to current epoch time, and `local_timezone` is non-empty.
- `uptime.total_seconds` is non-negative and consistent with `days/hours/minutes/seconds`.
- `system_info.uuid` is non-empty.
- `kernel_info.platform` is `android`.
- `memory_info.total_bytes`/`available_bytes`/`threshold_bytes` are non-negative.
- `users.uid` is parseable and non-negative.
- `cpu_info.cores` is parseable and non-negative.

### Architecture at a glance

```text
Managed config (server_url + enroll_secret + host_uuid)
  -> AgentApplication refreshes credentials
  -> ApiClient enrolls via /api/fleet/orbit/enroll (node key persisted)
  -> DistributedCheckinWorker polls /api/v1/osquery/distributed/read
  -> OsqueryQueryEngine runs SQL on Android-backed tables
  -> Results posted to /api/v1/osquery/distributed/write
```

### Current limitations

- SQL support is intentionally limited (basic `SELECT` + simple `WHERE`).
- Debug distributed polling interval is currently hardcoded to 15 seconds.
- Debug builds allow cleartext HTTP for local development; production should use managed secure config.

### Roadmap / next steps

- Make distributed polling interval configurable (managed config or build config).
- Add explicit platform validation updates for Android query targeting.
- Add more Android-specific tables and improve schema/docs coverage.
- Expand integration tests for distributed query loop behavior.

### Troubleshooting by symptom

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| Host not appearing in Fleet | Wrong `server_url` or enroll secret | Use `adb reverse`, set `FLEET_SERVER_URL=http://127.0.0.1:8080`, reinstall debug build |
| `localhost` does not work from phone | Phone resolves localhost to itself | Use `adb reverse tcp:8080 tcp:8080` or LAN IP |
| Query returns no rows | Unsupported table/columns or no local data | Start with `os_version` and `device_info` queries |
| No distributed logs | App not started after install/clear | Launch with `adb shell monkey -p com.fleetdm.agent -c android.intent.category.LAUNCHER 1` |
| Build/install fails | Missing execute bit or SDK/JDK setup | Run `chmod +x ./gradlew` and verify SDK/JDK requirements |

### Security model (summary)

- Enrollment uses managed config (`server_url`, `enroll_secret`, `host_uuid`) or debug fallback in debug builds only.
- Node key and API credentials are stored encrypted on-device using Android Keystore-backed encryption.
- Network access is restricted by build type; debug permits local development workflows, while production is expected to use secure managed configuration.
