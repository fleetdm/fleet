# Android Osquery Test Plan

## Purpose
Provide a practical, repeatable test plan for validating Android osquery behavior, enrollment reliability, and security controls before merge/release.

## Scope
This plan covers:
- Enrollment and re-enrollment behavior
- Distributed query read/execute/write loop
- SQL surface and table behavior
- Security controls in enrollment/distributed paths
- Real-device validation and regression checks

Out of scope:
- Full certificate/SCEP matrix (covered by dedicated certificate tests)
- Non-Android platforms

## Reference contract/oracle
- Contract + oracle: `android/000_BEHAVIOR_CONTRACT_AND_ORACLE.md`
- Key mappings used in this plan:
  - Enrollment/transport: O1, O2
  - Distributed loop: O3, O4
  - Table oracles: O5, O6, O7, O9, O10, O11
  - Security baseline: O12

## Test environments
- Local dev:
  - macOS + Android SDK + JDK 17+
  - Fleet server reachable at localhost/LAN
  - Android device connected by ADB
- CI/local unit:
  - JVM tests via Gradle
- Real device matrix (manual):
  - At least 2 Android versions
  - At least 2 OEMs if possible

## Entry criteria
- Branch builds locally
- Contract/oracle updated and reviewed
- Test device enrolled or ready to enroll

## Exit criteria
- All P0/P1 tests pass
- No open high-severity security issues
- Known minor risks documented in contract and PR QA

## Priority levels
- P0: Must pass for merge
- P1: Should pass before review sign-off
- P2: Nice-to-have confidence expansion

## Automated test suite (P0)

### A1. Enrollment + URL hardening
Command:
```bash
./gradlew :app:testDebugUnitTest --tests com.fleetdm.agent.ApiClientReenrollTest --console=plain --no-daemon
```
Expected:
- Re-enrollment on 401 works
- Non-401 does not re-enroll
- Identity change clears node key
- URL validation rules enforced

### A2. Time/uptime table oracle checks
Command:
```bash
./gradlew :app:testDebugUnitTest --tests com.fleetdm.agent.osquery.TimeAndUptimeTableTest --console=plain --no-daemon
```
Expected:
- `time` and `uptime` return one row
- numeric/time sanity checks pass

### A3. System/kernel/memory table oracle checks
Command:
```bash
./gradlew :app:testDebugUnitTest --tests com.fleetdm.agent.osquery.SystemKernelMemoryTableTest --console=plain --no-daemon
```
Expected:
- `system_info`, `kernel_info`, `memory_info` one-row shape and value parseability checks pass

### A4. Combined gate
Command:
```bash
./gradlew :app:compileDebugKotlin :app:testDebugUnitTest \
  --tests com.fleetdm.agent.ApiClientReenrollTest \
  --tests com.fleetdm.agent.osquery.TimeAndUptimeTableTest \
  --tests com.fleetdm.agent.osquery.SystemKernelMemoryTableTest \
  --console=plain --no-daemon
```
Expected:
- Full targeted suite passes

## Manual end-to-end validation (P0/P1)

### M1. Device enrollment smoke (P0)
Steps:
1. Confirm device:
```bash
adb devices
```
2. Reverse localhost:
```bash
adb reverse tcp:8080 tcp:8080
```
3. Install debug app:
```bash
FLEET_SERVER_URL='http://127.0.0.1:8080' ./gradlew installDebug
```
4. Reset app state and launch:
```bash
adb shell pm clear com.fleetdm.agent
adb shell monkey -p com.fleetdm.agent -c android.intent.category.LAUNCHER 1
```
Expected:
- Host appears in Fleet within expected check-in window

### M2. Distributed query loop smoke (P0)
Run in Fleet Live Query:
```sql
SELECT name, version, platform, security_patch FROM os_version;
```
Expected:
- Result returns from Android host
- No repeated stuck query behavior

### M3. Cross-OS table smoke (P1)
Run in Fleet Live Query:
```sql
SELECT * FROM time;
SELECT * FROM uptime;
SELECT * FROM system_info;
SELECT * FROM kernel_info;
SELECT * FROM memory_info;
```
Expected:
- Exactly one row per table
- Key sanity:
  - `time.unix_time` close to current epoch
  - `uptime.total_seconds >= 0`
  - `system_info.uuid` non-empty
  - `kernel_info.platform='android'`
  - memory numeric fields non-negative

### M4. Logcat table gating (P1)
1. Query with flag OFF:
```sql
SELECT * FROM android_logcat LIMIT 20;
```
Expected:
- Empty result
2. Enable managed config `enable_android_logcat_table=true` and retry.
Expected:
- Filtered Fleet-tag rows, no obvious secret leakage

## Security-focused tests (P0/P1)

### S1. URL policy (P0)
Verify behavior from automated tests (A1) and code paths:
- Non-debug requires HTTPS
- Reject path/query/fragment/userinfo in base URL

### S2. Re-enrollment control (P0)
Simulate/validate:
- 401 -> clear key -> re-enroll -> retry
- non-401 -> no re-enroll

### S3. Manifest security controls (P0)
Verify in manifest:
- `android:allowBackup="false"`
- `BootReceiver` exported false

### S4. Distributed path hardening (P0)
Code verification:
- `FleetDistributedQueryRunner` uses `ApiClient.distributedRead/distributedWrite`
- no custom unsafe TLS bypass path in active distributed loop

### S5. Identifier logging control (P1)
Release-like behavior:
- full Device ID not logged in non-debug

### S6. Accepted minor risk verification (P2)
- Enrollment secret currently in app-private DataStore (not Keystore-encrypted)
- Confirm risk remains documented in contract + PR QA

## Lifecycle and corner-case tests (P1/P2)

### L1. Host delete + re-enroll (P1)
Flow:
1. Enroll device
2. Delete host in Fleet UI
3. Trigger next check-in
Expected:
- 401 path re-enrolls and host recovers

### L2. App uninstall/reinstall (P1)
Expected:
- Fresh enroll works, no stale-key failure

### L3. Work profile reset (P2)
Expected:
- Re-enrollment and query loop recover after profile recreation

### L4. Network disruption (P2)
Cases:
- captive portal
- intermittent connectivity
- TLS interception/proxy
Expected:
- Failures are controlled, no crash loops, recovery when network normalizes

## Test reporting template
For each run, record:
- Build/commit
- Device model + Android version
- Tests run (IDs)
- Result: pass/fail
- Evidence: command output, screenshot, query result, log snippet
- Follow-ups/issues

## Recommended merge gate
Minimum for merge:
- A4 pass
- M1 + M2 pass on at least one real device
- S1/S2/S3/S4 pass
- Open issues triaged, high severity resolved
