---
name:  Release QA - fleetd
about: Checklist of required fleetd (Orbit, Fleet Desktop) and osquery tests prior to release
title: 'Release QA (fleetd):'
labels: '#g-orchestration,:release'
assignees: 'xpkoala,andreykizimenko,chrstphr84,Brajim20,marcusallen97,thisisjoegrant'

---

# Goal

Easy-to-follow test steps for manually checking the `fleetd` agent and the bundled osquery (`osqueryd`) prior to a release.

> **How to check off:** Tick the checkbox in each **Progress** list as a test passes. GitHub tracks completion ("X of Y tasks") at the top of the issue. The tables below each list hold the step instructions and expected results for reference. For a **failure**, leave the box unchecked and record the details under [Notes](#notes).

# Important reference data

1. [fleetctl preview setup](https://fleetdm.com/fleetctl-preview)
2. [Testing TUF](https://github.com/fleetdm/fleet/blob/main/tools/tuf/test/README.md)
3. [`fleetd` auto-update n+1 test guide](https://github.com/fleetdm/fleet/blob/main/tools/tuf/test/Fleetd-auto-update-test-guide.md)

# Release versioning

Includes updates to:
<!-- Remove items without updates -->
- Orbit: True / False : `v1.xx.x` > `v1.xx.x`
- Desktop: True / False : `v1.xx.x` > `v1.xx.x`
- osquery: True / False : `v1.xx.x` > `v1.xx.x`
- Android: True / False : `v1.xx.x` > `v1.xx.x`

# Smoke tests

Smoke tests are limited to core functionality and serve as a pre-release final review. If smoke
tests are failing, a release cannot proceed.

> **Setup:** Before running, build a new `fleetd` from the release candidate branch as needed for Orbit and Desktop (e.g. `rc-minor-fleet-v4.80.0`). Do not build from `main` — it is a moving target and changes from future releases may already be merged.

## fleetd

### Prerequisites

1. Local instance is running and up to date with the target release branch.
2. `fleetd` is built from the release candidate branch (e.g. `rc-minor-fleet-v4.xx.0`), not `main`.
3. Hosts enrolled for macOS, Windows, and Linux.

**Progress**
- [ ] Enrollment & versions
- [ ] Fleet Desktop / My device page
- [ ] Scripts
- [ ] Software
- [ ] Auto-updates disabled
- [ ] Auto-update n+1
- [ ] Self-healing

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>


<tr>
<td>Enrollment & versions</td>
<td>Verify hosts enroll and report correct component versions.</td>
<td>

1. Hosts can enroll and report the correct version of `fleetd` (Orbit, osquery, Fleet Desktop) on macOS, Windows, and Linux.
2. Refetching host vitals completes and returns updated information.

</td>
</tr>

<tr>
<td>Fleet Desktop / My device page</td>
<td>Verify all sections of the end user's My device page load and display correctly.</td>
<td>

1. Clicking the Fleet desktop item, then "My device" successfully loads the My device page.
2. All tabs/sections of the "My device" page (e.g. About, Software, Activity) are populated correctly and as expected.
3. Self-service software items are visible and accessible under the Software tab.
4. Styling and padding appears correct across all sections.

</td>
</tr>

<tr>
<td>Scripts</td>
<td>Verify script execution via Orbit.</td>
<td>

1. From Host details (macOS, Windows, & Linux) run a script that should PASS, verify.
2. From Host details (macOS, Windows, & Linux) run a script that should FAIL, verify.
3. Verify script results display correctly in Activity feed.

</td>
</tr>

<tr>
<td>Software</td>
<td>Verify software install via Orbit.</td>
<td>

1. From Host details (macOS, Windows, & Linux) run an install that should PASS, verify.
2. From My Device (macOS, Windows, & Linux) software tab should have self-service items available, verify.
3. Verify software installs display correctly in Activity feed.

</td>
</tr>

<tr>
<td>Self-healing</td>
<td>Verify Orbit restarts automatically after being stopped.</td>
<td>

1. Stop or kill the Orbit process on a macOS, Windows, and Linux host.
2. Wait and confirm the Orbit process restarts automatically without manual intervention.
3. Confirm the host remains enrolled and visible in Fleet after the restart.

</td>
</tr>

<tr>
<td>Auto-updates disabled</td>
<td>Verify that fleetd works when the installer package is built with <code>--disable-updates</code>.</td>
<td>

1. Generate package with `fleetctl package [...] --updates-disabled`.
2. Install packages on macOS, Windows, and Linux.
3. Smoke test Orbit and Fleet Desktop functionality, and osquery tables.

</td>
</tr>

<tr>
<td>Auto-update n+1</td>
<td>Verify the agent successfully auto-updates to the new release.</td>
<td>

1. Conduct the [`fleetd` auto-update n+1 test](https://github.com/fleetdm/fleet/blob/main/tools/tuf/test/Fleetd-auto-update-test-guide.md).
2. Agent successfully auto-updates.
3. QA certifies new release by commenting in issue.

</td>
</tr>

</table>

## osquery

**Progress**
- [ ] osquery version
- [ ] Live query (report) flow
- [ ] osquery tables
- [ ] Scheduled queries & log destination
- [ ] Packs flow

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>


<tr>
<td>osquery version</td>
<td>Verify the bundled osquery version.</td>
<td>

1. Host details page reports the expected `osqueryd` version on macOS, Windows, and Linux, matching the version listed in Release versioning above.

</td>
</tr>

<tr>
<td>Live query (report) flow</td>
<td>Run a live query against enrolled hosts.</td>
<td>

1. A live query (e.g. `SELECT * FROM osquery_info;`) runs manually and returns results from macOS, Windows, and Linux hosts.
2. Syntax errors result in error messaging.

</td>
</tr>

<tr>
<td>osquery tables</td>
<td>Verify common osquery tables return data on each platform.</td>
<td>

1. Query platform-relevant tables (e.g. `system_info`, `os_version`, `users`, `processes`) and confirm results are populated on macOS, Windows, and Linux.

</td>
</tr>

<tr>
<td>Scheduled queries & log destination</td>
<td>Verify scheduled query execution and result logging.</td>
<td>

1. Scheduled queries run on host machines and results are logged.
2. osquery result and status logs are successfully sent to the configured log destination (Filesystem or external).

</td>
</tr>

<tr>
<td>Packs flow</td>
<td>Verify management, operation, and logging of ["2017 packs"](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs).</td>
<td>

1. Packs successfully run on host machines after migrations.
2. New Packs can be created.
3. Packs can be edited and deleted.
4. Packs results information is logged.

</td>
</tr>

</table>

## Android

**Progress**
- [ ] App deployment
- [ ] Setup experience software
- [ ] Software available in Google Play
- [ ] Certificate delivery
- [ ] Certificate display in app
- [ ] Debug mode
- [ ] Unenrollment

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>


<tr>
<td>App deployment</td>
<td>Verify the Fleet Android app is deployed to the device.</td>
<td>

1. The Fleet Android app gets deployed on the device.

</td>
</tr>

<tr>
<td>Setup experience software</td>
<td>Verify setup experience software is pre-installed during enrollment.</td>
<td>

1. Setup experience software gets pre-installed during the enrollment.

</td>
</tr>

<tr>
<td>Software available in Google Play</td>
<td>Verify newly added software is available to the end user.</td>
<td>

1. Newly added software is available in Google Play.

</td>
</tr>

<tr>
<td>Certificate delivery</td>
<td>Verify certificates can be delivered to the device.</td>
<td>

1. Certificates can be delivered.

</td>
</tr>

<tr>
<td>Certificate display in app</td>
<td>Verify certificates are displayed correctly within the Fleet Android app.</td>
<td>

1. Certificates are displayed correctly within the Fleet Android app.

</td>
</tr>

<tr>
<td>Debug mode</td>
<td>Verify debug mode can be accessed from the app.</td>
<td>

1. Debug mode can be accessed by tapping the app version 10 times.

</td>
</tr>

<tr>
<td>Unenrollment</td>
<td>Verify an Android device can be unenrolled from Fleet.</td>
<td>

1. Unenroll the device from the Fleet UI.
2. Confirm the device is removed from the host list in Fleet.
3. Confirm the Fleet Android app is no longer managed on the device.

</td>
</tr>

</table>

## Notes

What has not been tested:

Include any notes on whether issues should block release or not as needed:
