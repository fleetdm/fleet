---
name:  Release QA
about: Checklist of required tests prior to release
title: 'Release QA:'
labels: '#g-orchestration,#g-apple-at-work,#g-power-to-pc,#g-auto-patching,#g-security-compliance,:release'
assignees: 'xpkoala,andreykizimenko,chrstphr84,Brajim20,marcusallen97,thisisjoegrant'

---

# Goal

Easy-to-follow test steps for checking a release manually.

> **How to check off:** Tick the checkbox in each **Progress** list as a test passes. GitHub tracks completion ("X of Y tasks") at the top of the issue. The tables below each list hold the step instructions and expected results for reference. For a **failure**, leave the box unchecked and record the details under the **Notes** section at the bottom of this issue.

# Important reference data

1. [fleetctl preview setup](https://fleetdm.com/fleetctl-preview)
2. [Permissions documentation](https://fleetdm.com/docs/using-fleet/permissions)
3. [Fleet free vs premium documentation](https://fleetdm.com/pricing)

# Smoke tests

Smoke tests are limited to core functionality and serve as a pre-release final review. If smoke tests are failing, a release cannot proceed.

## Fleet core

**Fleet version** (Head to the "My account" page in the Fleet UI or run `fleetctl version`):

**Web browser** _(e.g. Chrome 88.0.4324)_:

### Prerequisites

1. Local instance is running and up to date with the target release branch.
2. In your browser, clear local storage using devtools.

### Orchestration

**Progress**
- [ ] Update flow
- [ ] Login flow
- [ ] Packs flow
- [ ] Log destination flow
- [ ] IdP Provisioning (SCIM)
- [ ] GitOps and generate-gitops
- [ ] Fleet Free

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>Update flow</td>
<td>

1. Remove all fleet processes/agents/etc using `fleetctl preview reset` for a clean slate.
2. Run `fleetctl preview` with no tag for latest stable.
3. Create a host/report to later confirm upgrade with.
4. STOP fleet-preview-server instances in containers/apps on Docker.
5. Run `fleetctl preview` with appropriate testing tag.
6. Navigate through all new UI flows and confirm dashboard, hosts, controls, queries, policies, and settings pages are working as expected.

</td>
<td>All previously created hosts/queries are verified to still exist.</td>
</tr>

<tr>
<td>Login flow</td>
<td>

1. Navigate to the login page and attempt to login with both valid and invalid credentials to verify some combination of expected results.
2. Navigate to the login page and attempt to login with both valid and invalid SSO credentials to verify expected results.

</td>
<td>

1. Text fields prompt when blank.
2. Correct error message is "authentication failed".
3. Forget password link prompts for email.
4. Valid credentials result in a successful login.
5. Valid SSO credentials result in a successful login.

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

<tr>
<td>Log destination flow</td>
<td>Verify log destination for software, reports, policies, and packs.</td>
<td>

1. Software, report, policy, and packs logs are successfully sent to external log destinations.
2. Software, report, policy, and packs logs are successfully sent to Filesystem log destinations.

</td>
</tr>

<tr>
<td>IdP Provisioning (SCIM)</td>
<td>Verify host vitals sync.</td>
<td>

1. Configure and verify provisioning with the following IdPs:
    1. Okta
    2. Entra
    3. Hydrant/Google
2. Enroll hosts with EUA & IdP Provisioning enabled:
    1. macOS
    2. Windows
    3. Ubuntu
    4. iOS/iPadOS
    5. Android

</td>
</tr>

<tr>
<td>GitOps and generate-gitops</td>
<td>Verify <code>fleetctl generate-gitops</code> and GitOps functionality.</td>
<td>

1. Generate-gitops from a version-matched fleetctl successfully outputs YAML from a brand new Fleet server (net of auto-populated fleets etc.).
2. Running GitOps either using the `gitops.sh` script directly (from the `fleet-gitops` repo) or by using the GitOps GitHub or GitLab workflow (attempting via one of these three is sufficient) succeeds.

</td>
</tr>

<tr>
<td>Fleet Free</td>
<td>Verify that product group features behave correctly on Fleet Free.</td>
<td>

Run basic checks for the product group area while using a Fleet Free license.

- Features documented as Free work normally:
   - Packs
   - GitOps
- Premium features are correctly restricted or hidden:
   - IdP information
- No UI, API, or workflow errors occur when using Free-only functionality.

Reference: https://fleetdm.com/pricing

</td>
</tr>

</table>

### Apple at Work

**Progress**
- [ ] MDM enrollment flow
- [ ] MDM migration flow
- [ ] OS settings
- [ ] Disk encryption
- [ ] OS updates
- [ ] Setup experience
- [ ] iOS/iPadOS
- [ ] Token & Certificate Renewals
- [ ] Fleet Free

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>MDM enrollment flow</td>
<td>Verify MDM enrollments, run MDM commands.</td>
<td>

1. Erase an ADE-eligible macOS host and verify able to complete automated enrollment flow.
2. Verify able to run MDM commands on macOS hosts from the CLI.

</td>
</tr>

<tr>
<td>MDM migration flow</td>
<td>Verify MDM migration for ADE and non-ADE hosts.</td>
<td>

1. Turn off MDM on an ADE-eligible macOS host and verify that the native, "Device Enrollment" macOS notification appears.
2. On the My device page, follow the "Turn on MDM" instructions and verify that MDM is turned on.
3. Turn off MDM on a non ADE-eligible macOS host.

</td>
</tr>

<tr>
<td>OS settings</td>
<td>Verify OS settings functionality.</td>
<td>

1. Verify Profiles upload/download/delete.
2. Verify Profiles are delivered to host and applied.

</td>
</tr>

<tr>
<td>Disk encryption</td>
<td>Verify disk encryption functionality (macOS).</td>
<td>

1. Verify able to configure Disk encryption (macOS).
2. Verify host enrolled with Disk encryption enforced successfully encrypts.

</td>
</tr>

<tr>
<td>OS updates</td>
<td>Verify OS updates flow (macOS).</td>
<td>

1. Configure OS updates (macOS).
2. Verify enforce minimumOS occurs during enrollment (macOS 14+).

</td>
</tr>

<tr>
<td>Setup experience</td>
<td>Verify macOS Setup experience.</td>
<td>

1. Configure End user authentication.
2. Upload a Bootstrap package.
3. Add software (FMA, VPP, & Custom pkg).
4. Add a script.
5. Enroll an ADE-eligible macOS host and verify successful authentication.
6. Verify Bootstrap package is delivered.
7. Verify SwiftDialogue window displays.
8. Verify software installs and script runs.

</td>
</tr>

<tr>
<td>iOS/iPadOS</td>
<td>Verify enrollment, profiles, & software installs.</td>
<td>

1. Verify ADE enrollment.
2. Verify BYOD OTA enrollment.
3. Verify BYOD Account-driven user enrollment (AppleID).
4. Verify Profiles are delivered to host and applied.
5. Verify VPP apps install & display correctly in Activity feed.
6. Verify `Turn Off MDM` for BYOD & ADE hosts.

</td>
</tr>

<tr>
<td>Token & Certificate Renewals</td>
<td>APNs cert and ABM token renewal workflow.</td>
<td>

1. Renew APNs Certificate.
2. Renew ABM Token.
3. Ensure ADE hosts can enroll.

</td>
</tr>

<tr>
<td>Fleet Free</td>
<td>Verify that product group features behave correctly on Fleet Free.</td>
<td>

Run basic checks for the product group area while using a Fleet Free license.

- Features documented as Free work normally:
   - Host enrollment
   - Apple MDM
   - Configuration profile delivery
   - APNs Certificate renewal
- Premium features are correctly restricted or hidden:
   - Disk encryption
   - OS updates
   - Setup experience
- No UI, API, or workflow errors occur when using Free-only functionality.

Reference: https://fleetdm.com/pricing

</td>
</tr>

</table>

### Power to PC

**Progress**
- [ ] MDM enrollment flow
- [ ] MDM migration flow
- [ ] OS settings
- [ ] Disk encryption
- [ ] OS updates
- [ ] Setup Experience
- [ ] Android
- [ ] Fleet Free

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>MDM enrollment flow</td>
<td>Verify MDM enrollments, run MDM commands.</td>
<td>

1. With Windows MDM turned On, enroll a Windows host and verify MDM is turned On for the host.
2. Erase an Auto-Pilot enabled Windows host and complete automated enrollment flow.
3. Verify able to run MDM commands on Windows hosts from the CLI.

</td>
</tr>

<tr>
<td>MDM migration flow</td>
<td>Verify MDM migration for Windows hosts.</td>
<td>

1. Verify Windows host migrates from 3rd party MDM to Fleet when automatic migration is turned on.

</td>
</tr>

<tr>
<td>OS settings</td>
<td>Verify OS settings functionality.</td>
<td>

1. Verify Profiles upload/download/delete.
2. Verify Profiles are delivered to host and applied.

</td>
</tr>

<tr>
<td>Disk encryption</td>
<td>Verify disk encryption functionality (Windows).</td>
<td>

1. Verify able to configure Disk encryption (Windows).
2. Verify host enrolled with Disk encryption enforced successfully encrypts.

</td>
</tr>

<tr>
<td>OS updates</td>
<td>Verify OS updates flow (Windows).</td>
<td>

1. Configure OS updates (Windows).

</td>
</tr>

<tr>
<td>Setup Experience</td>
<td>Verify Windows Setup experience.</td>
<td>

1. Configure End user authentication.
2. Add software (FMA, Custom pkg).

</td>
</tr>

<tr>
<td>Android</td>
<td>Verify enrollment, profiles, & software installs.</td>
<td>

1. Verify BYOD enrollment.
2. Verify Profiles are delivered to host and applied.
3. Verify apps install.
4. Verify certificate delivery.
5. Verify `Unenroll`.

</td>
</tr>

<tr>
<td>Fleet Free</td>
<td>Verify that product group features behave correctly on Fleet Free.</td>
<td>

Run basic checks for the product group area while using a Fleet Free license.

- Features documented as Free work normally:
   - Host enrollment
   - Windows MDM
   - Android MDM
   - Configuration profile delivery
- Premium features are correctly restricted or hidden:
   - Automatic MDM migration
   - Disk encryption
   - OS updates
- No UI, API, or workflow errors occur when using Free-only functionality.

Reference: https://fleetdm.com/pricing

</td>
</tr>

</table>

### Auto Patching

**Progress**
- [ ] Report flow
- [ ] Host flow
- [ ] My device page
- [ ] Scripts
- [ ] Software
- [ ] Fleet Free

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>Report flow</td>
<td>Create, edit, run, and delete reports.</td>
<td>

1. Permissions regarding creating/editing/deleting reports are up to date with documentation.
2. Syntax errors result in error messaging.
3. Queries can be run manually.

</td>
</tr>

<tr>
<td>Host flow</td>
<td>Verify a new host can be added and removed following modal instructions using your own device.</td>
<td>

1. Host is added via command line.
2. Host serial number and date added are accurate.
3. Host is not visible after it is deleted.
4. Warning and informational modals show when expected and make sense.

</td>
</tr>

<tr>
<td>My device page</td>
<td>Verify the end user's My device page loads successfully.</td>
<td>

1. Clicking the Fleet desktop item, then "My device" successfully loads the My device page.
2. The "My device" page is populated correctly and as expected.
3. Styling and padding appears correct.

</td>
</tr>

<tr>
<td>Scripts</td>
<td>Verify script library and execution.</td>
<td>

1. Verify able to run a script on all host types from CLI.
2. Verify scripts library upload/download/delete.
3. From Host details (macOS, Windows, & Linux) run a script that should PASS, verify.
4. From Host details (macOS, Windows, & Linux) run a script that should FAIL, verify.
5. Verify UI loading state and statuses for scripts.
6. Disable scripts globally and verify unable to run.
7. Verify scripts display correctly in Activity feed.

</td>
</tr>

<tr>
<td>Software</td>
<td>Verify software library and install / download.</td>
<td>

1. Verify software library upload/download/delete.
2. From Host details (macOS, Windows, & Linux) run an install that should PASS, verify.
3. From My Device (macOS, Windows, & Linux) software tab should have self-service items available, verify.
4. Verify UI loading state and statuses for installing software.
5. Verify software installs display correctly in Activity feed.

</td>
</tr>

<tr>
<td>Fleet Free</td>
<td>Verify that product group features behave correctly on Fleet Free.</td>
<td>

Run basic checks for the product group area while using a Fleet Free license.

- Features documented as Free work normally:
   - Host details page
   - Reports (Add, edit, live report)
   - Software inventory
   - Scripts (Add, delete, run)
   - My device page (Mac, Windows, Linux)
- Premium features are correctly restricted or hidden:
   - Add software
- No UI, API, or workflow errors occur when using Free-only functionality.

Reference: https://fleetdm.com/pricing

</td>
</tr>

</table>

### Security & Compliance

**Progress**
- [ ] Disk encryption (Linux)
- [ ] Vulnerabilities
- [ ] Certificate Authorities
- [ ] Lock & Wipe
- [ ] Fleet Free

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>Disk encryption (Linux)</td>
<td>Verify disk encryption functionality (Linux).</td>
<td>

1. Verify able to configure Disk encryption (Linux).
2. Verify host enrolled with Disk encryption enforced successfully encrypts.

</td>
</tr>

<tr>
<td>Vulnerabilities</td>
<td>Verify that software vulnerabilities are correctly populated.</td>
<td>

1. Verify that known vulnerable software items display expected CVEs and severity information in the Software tab.
2. Verify that individual vulnerabilities can be previewed and open the correct NVD page when selected.
3. Verify that vulnerable software appears under "My device > Software" for affected hosts with expected CVEs.

</td>
</tr>

<tr>
<td>Certificate Authorities</td>
<td>Verify setup and certificate delivery.</td>
<td>

1. Configure and verify that certificates deploy to hosts with the following CAs:
    1. DigiCert
    2. NDES
    3. SmallStep

</td>
</tr>

<tr>
<td>Lock & Wipe</td>
<td>Verify hosts can be locked & wiped.</td>
<td>

1. Verify locking a host from the Fleet UI (macOS, Windows, & Linux).
2. Verify unlocking a host from the Fleet UI (macOS, Windows, & Linux).
3. Verify wiping a host from the Fleet UI (macOS, Windows, & Linux).
4. Verify wiping and locking hosts using `fleetctl` (macOS, Windows, & Linux).

</td>
</tr>

<tr>
<td>Fleet Free</td>
<td>Verify that product group features behave correctly on Fleet Free.</td>
<td>

Run basic checks for the product group area while using a Fleet Free license.

- Features documented as Free work normally:
   - Vulnerability detection
   - Individual CVE page
- Premium features are correctly restricted or hidden:
   - Disk encryption (Linux)
   - Lock / Wipe
   - Certificate authorities
- No UI, API, or workflow errors occur when using Free-only functionality.

Reference: https://fleetdm.com/pricing

</td>
</tr>

</table>

### All product groups

**Progress**
- [ ] Release-critical issues (Ready for release)
- [ ] Baseline loadtest (minor releases only)
- [ ] Migration loadtest (minor releases only)
- [ ] Helm chart
- [ ] Migration test
- [ ] Cloud migration tests
- [ ] Trivy scan

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th></tr>

<tr>
<td>Release-critical issues (Ready for release)</td>
<td>Verify no open <code>~unreleased bug</code> or <code>~release blocker</code> issue is still in the works — all should be "Ready for release".</td>
<td>

1. Check the [`~unreleased bug`](https://github.com/fleetdm/fleet/labels/~unreleased%20bug) and [`~release blocker`](https://github.com/fleetdm/fleet/labels/~release%20blocker) filters and confirm every open issue is "Ready for release" on its product group board.
2. If any is not, raise an alarm in the `#help-engineering` and the relevant product group channel.

</td>
</tr>

<tr>
<td>Baseline loadtest - minor releases only unless otherwise specified</td>
<td>Verify load test metrics are within acceptable range on a fresh RC instance with no data, compared against the previous release.</td>
<td>

1. After all expected changes have been merged to the RC branch, set up a load test environment using the RC branch (new instance, no data) and allow it at least 24hrs of run time.
2. Collect metrics with the [`collect-metrics.sh`](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/collect-metrics.sh) script under the `baseline` category (see the [metrics README](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/README.md)), e.g. `./collect-metrics.sh --workspace <version>loadtest --category baseline`.
3. Compare the run against the previous release (n-1) with [`compare-metrics.sh`](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/compare-metrics.sh) and post the comparison output as a comment on this issue. All deltas should report `ok` — investigate any `WARN`/`ALERT`.
4. Open a PR against `main` with the run artifacts (`.json` + `.md`) under `runs/baseline/<workspace>/`, and record the metrics in [this spreadsheet](https://docs.google.com/spreadsheets/d/1FOF0ykFVoZ7DJSTfrveip0olfyRQsY9oT1uXCCZmuKc/edit?usp=drive_link).

</td>
</tr>

<tr>
<td>Migration loadtest - minor releases only unless otherwise specified</td>
<td>Verify load test metrics hold steady after migrating from the previous minor release (n-1) to the RC (n), comparing before vs. after the migration.</td>
<td>

1. Run a load test on the previous minor release (n-1). Right before migrating, collect the last 2 hours with [`collect-metrics.sh`](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/collect-metrics.sh) under the `migration` category (see the [metrics README](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/README.md)), e.g. `./collect-metrics.sh --workspace <n-1>to<n>mig --category migration --interval 2h`.
2. Follow [Deploying code changes to fleet](https://github.com/fleetdm/fleet/blob/main/infrastructure/loadtesting/terraform/readme.md#deploying-code-changes-to-fleet) to migrate the environment to the RC branch (n).
3. Wait ~2 hours, then collect the past 2 hours with `collect-metrics.sh`, e.g. `./collect-metrics.sh --workspace <n-1>to<n>mig --category migration --interval 2h`.
4. Compare the post-migration run against the pre-migration run with [`compare-metrics.sh`](https://github.com/fleetdm/fleet/blob/main/tools/loadtest/metrics/compare-metrics.sh) and post the comparison output as a comment on this issue. All deltas should report `ok` — investigate any `WARN`/`ALERT`.
5. Open a PR against `main` with the run artifacts (`.json` + `.md`) under `runs/migration/<workspace>/`, and record the metrics in [this spreadsheet](https://docs.google.com/spreadsheets/d/1FOF0ykFVoZ7DJSTfrveip0olfyRQsY9oT1uXCCZmuKc/edit?usp=drive_link).

</td>
</tr>

<tr>
<td>Helm chart</td>
<td>Verify the Fleet Helm chart deploys cleanly against the RC image.</td>
<td>

Follow the [Test Fleet Helm Chart With Docker Desktop runbook](https://github.com/fleetdm/confidential/blob/main/infrastructure/runbooks/test-fleet-helm-chart-with-docker-desktop.md).

1. Back up your local Fleet dev MySQL data if you have anything you care about
2. Start MySQL and Redis from the Fleet repo's Docker Compose file.
3. Prepare the `fleet` namespace and `mysql` secret on the `docker-desktop` kube context.
4. `helm upgrade --install` the chart from `./charts/fleet` using the RC image tag (e.g. `imageTag=rc-minor-fleet-v4.xx.0`).
5. Confirm the Fleet pod becomes Ready (`kubectl -n fleet get pods`) with no errors in `kubectl -n fleet logs deploy/fleet -c fleet`.
6. Port-forward `svc/fleet-service` and confirm the Fleet UI loads at `http://localhost:8080`.
7. Attach a screenshot of `kubectl -n fleet get pods` (showing the Fleet pod Ready) and the loaded Fleet UI to this issue as a comment.
8. Tear down with `helm uninstall` + `kubectl delete namespace fleet` + `docker compose stop mysql redis`.

</td>
</tr>

<tr>
<td>Migration test</td>
<td>Verify Fleet can migrate to the next version with no issues.</td>
<td>

Using [this GitHub action](https://github.com/fleetdm/fleet/actions/workflows/db-upgrade-test.yml):

1. Using the most recent stable version of Fleet and `main`, click `Run workflow`.
2. Enter the Docker tag of Fleet starting version, e.g. `v4.64.2`.
3. Enter the Docker tag of Fleet version to upgrade to, e.g. `rc-minor-fleet-v4.65.0`.
4. Click `Run workflow`.
5. Action should complete successfully.

</td>
</tr>

<tr>
<td>Cloud migration tests</td>
<td>Verify Fleet can migrate when using real world data.</td>
<td>

Using [this GitHub action](https://github.com/fleetdm/confidential/actions/workflows/cloud-tests.yml):

1. Enter `fleetdm/fleet:rc-minor-fleet-<version>` for `The image to test`.
2. Select `all` for `Where will we deploy?`.
3. Action should complete successfully and the total time for each instance shouldn't be drastically different from previous releases.

</td>
</tr>

<tr>
<td>Trivy scan</td>
<td>Verify the latest Trivy scan of the RC image has no new high/critical vulnerabilities.</td>
<td>

1. Using [this GitHub action](https://github.com/fleetdm/fleet/actions/workflows/trivy-scan.yml).
2. Review the scan results for any new high or critical severity vulnerabilities introduced by this RC.
3. Attach a screenshot of the latest Trivy scan results to this issue as a comment.
4. If new high/critical vulnerabilities are found, raise an alarm in the appropriate channels.

</td>
</tr>

</table>

## Notes

Issues found new to this version:

Issues found that reproduce in last stable version:

What has not been tested:

Include any notes on whether issues should block release or not as needed:
