---
name:  Release QA
about: Checklist of required tests prior to release
title: 'Release QA:'
labels: '#g-mdm,#g-orchestration,#g-software,:release'
assignees: 'xpkoala,pezhub,jmwatts'

---

# Goal: easy-to-follow test steps for checking a release manually

# Important reference data

1. [fleetctl preview setup](https://fleetdm.com/fleetctl-preview)
2. [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) 
3. premium tests require license key (needs renewal) `fleetctl preview --license-key=eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjQwOTk1MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjIyNDI2NTg2fQ.WmZ0kG4seW3IrNvULCHUPBSfFdqj38A_eiXdV_DFunMHechjHbkwtfkf1J6JQJoDyqn8raXpgbdhafDwv3rmDw`
4. premium tests require license key (active - Expires Sunday, January 1, 2023 12:00:00 AM) `fleetctl preview --license-key=eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjcyNTMxMjAwLCJzdWIiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6InByZW1pdW0iLCJpYXQiOjE2NDI1MjIxODF9.EGHQjIzM73YyMbnCruswzg360DEYCsDi9uz48YcDwQHq90BabGT5PIXRiculw79emGj5sk2aKgccTd2hU5J7Jw`

# Smoke Tests
Smoke tests are limited to core functionality and serve as a pre-release final review. If smoke tests are failing, a release cannot proceed.

## Fleet core:

**Fleet version** (Head to the "My account" page in the Fleet UI or run `fleetctl version`):

**Web browser** _(e.g. Chrome 88.0.4324)_: 

### Prerequisites

1. `fleetctl preview` is set up and running the desired test version using [`--tag` parameters.](https://fleetdm.com/handbook/engineering#run-fleet-locally-for-qa-purposes)
2. Unless you are explicitly testing older browser versions, browser is up to date.
3. Certificate & flagfile are in place to create new host.
4. In your browser, clear local storage using devtools.

### Orchestration
<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>Update flow</td><td>

1. remove all fleet processes/agents/etc using `fleetctl preview reset` for a clean slate
2. run `fleetctl preview` with no tag for latest stable
3. create a host/query to later confirm upgrade with
4. STOP fleet-preview-server instances in containers/apps on Docker
5. run `fleetctl preview` with appropriate testing tag </td><td>All previously created hosts/queries are verified to still exist</td><td>pass/fail</td></tr>
<tr><td>Login flow</td><td>

1. navigate to the login page and attempt to login with both valid and invalid credentials to verify some combination of expected results.
2. navigate to the login page and attempt to login with both valid and invalid sso credentials to verify expected results.
</td><td>

1. text fields prompt when blank
2. correct error message is "authentication failed"
3. forget password link prompts for email
4. valid credentials result in a successful login.
5. valid sso credentials result in a successful login</td><td>pass/fail</td></tr>
<tr><td>Packs flow</td><td>Verify management, operation, and logging of ["2017 packs"](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs).</td><td>

1. Packs successfully run on host machines after migrations 
2. New Packs can be created
3. Packs can be edited and deleted
4. Packs results information is logged
 
</td><td>pass/fail</td></tr>

<tr><td>Log destination flow</td><td>Verify log destination for software, query, policy, and packs.</td><td>

1. Software, query, policy, and packs logs are successfully sent to external log destinations
2. Software, query, policy, and packs logs are successfully sent to Filesystem log destinations
 
</td><td>pass/fail</td></tr>
<tr><td>OS settings</td><td>Verify OS settings functionality</td><td>

1. Verify able to configure Disk encryption (macOS, Windows, & Linux).
2. Verify host enrolled with Disk encryption enforced successfully encrypts.
</td><td>pass/fail</td></tr>
</table>

### MDM
<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>MDM enrollment flow</td><td>Verify MDM enrollments, run MDM commands</td><td>
  
1. Erase an ADE-eligible macOS host and verify able to complete automated enrollment flow.
2. With Windows MDM turned On, enroll a Windows host and verify MDM is turned On for the host.
3. Verify able to run MDM commands on both macOS and Windows hosts from the CLI.
</td><td>pass/fail</td></tr>

<tr><td>MDM migration flow</td><td>Verify MDM migration for ADE and non-ADE hosts</td><td>
  
1. Turn off MDM on an ADE-eligible macOS host and verify that the native, "Device Enrollment" macOS notification appears.
2. On the My device page, follow the "Turn on MDM" instructions and verify that MDM is turned on.
3. Turn off MDM on a non ADE-eligible macOS host.
4. On the My device page, follow the "Turn on MDM" instructions and verify that MDM is turned on.
</td><td>pass/fail</td></tr>
<tr><td>OS settings</td><td>Verify OS settings functionality</td><td>

1. Verify Profiles upload/download/delete (macOS & Windows).
2. Verify Profiles are delivered to host and applied. 
</td><td>pass/fail</td></tr>

<tr><td>Setup experience</td><td>Verify macOS Setup experience</td><td>

1. Configure End user authentication.
3. Upload a Bootstrap package.
4. Add software (FMA, VPP, & Custom pkg)
5. Add a script
6. Enroll an ADE-eligible macOS host and verify successful authentication.
7. Verify Bootstrap package is delivered.
8. Verify SwiftDialogue window displays.
9. Verify software installs and script runs.
</td><td>pass/fail</td></tr>

<tr><td>OS updates</td><td>Verify OS updates flow</td><td>

1. Configure OS updates (macOS & Windows).
2. Verify on-device that Nudge prompt appears (macOS 13).
3. Verify enforce minimumOS occurs during enrollment (macOS 14+).
</td><td>pass/fail</td></tr>

<tr><td>iOS/iPadOS</td><td>Verify enrollment, profiles, & software installs</td><td>

1. Verify ADE enrollment.
2. Verify OTA enrollment.
3. Verify Profiles are delivered to host and applied.
4. Verify VPP apps install & display correctly in Activity feed.
 
</td><td>pass/fail</td></tr>
<tr><td>Certificates Upload</td><td>APNs cert and ABM token renewal workflow</td><td>

1. Renew APNs Certificate.
2. Renew ABM Token.
3. Ensure ADE hosts can enroll.
</td><td>pass/fail</td></tr>

</table>

### Software
<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>Query flow</td><td>Create, edit, run, and delete queries. </td><td>

1. permissions regarding creating/editing/deleting queries are up to date with documentation
2. syntax errors result in error messaging
3. queries can be run manually 
</td><td>pass/fail</td></tr>
<tr><td>Host Flow</td><td>Verify a new host can be added and removed following modal instructions using your own device.</td><td>

1. Host is added via command line
2. Host serial number and date added are accurate
3. Host is not visible after it is deleted
4. Warning and informational modals show when expected and make sense
</td><td>pass/fail</td></tr>
<tr><td>My device page</td><td>Verify the end user's my device page loads successfully.</td><td>

1. Clicking the Fleet desktop item, then "My device" successfully loads the my device page.
2. The "My device" page is populated correctly and as expected. 
3. Styling and padding appears correct.
 
</td><td>pass/fail</td></tr>
<tr><td>Scripts</td><td>Verify script library and execution</td><td>

1. Verify able to run a script on all host types from CLI.
2. Verify scripts library upload/download/delete.
3. From Host details (macOS, Windows, & Linux) run a script that should PASS, verify.
4. From Host details (macOS, Windows, & Linux) run a script that should FAIL, verify.
5. Verify UI loading state and statuses for scripts.
8. Disable scripts globally and verify unable to run.
9. Verify scripts display correctly in Activity feed.
</td><td>pass/fail</td></tr>

<tr><td>Software</td><td>Verify software library and install / download</td><td>

1. Verify software library upload/download/delete.
2. From Host details (macOS, Windows, & Linux) run an install that should PASS, verify.
3. From My Device (macOS, Windows, & Linux) software tab should have self-service items available, verify.
4. Verify UI loading state and statuses for installing software.
7. Verify software installs display correctly in Activity feed.
</td><td>pass/fail</td></tr>


<tr><td>Migration Test</td><td>Verify Fleet can migrate to the next version with no issues.</td><td>

Using the github action https://github.com/fleetdm/fleet/actions/workflows/db-upgrade-test.yml
1. Using the most recent stable version of Fleet and `main`, click `Run workflow`
2. Enter the Docker tag of Fleet starting version, e.g. 'v4.64.2'
3. Enter the Docker tag of Fleet version to upgrade to, e.g. 'rc-minor-fleet-v4.65.0'
4. Click `Run workflow`.
5. Action should complete successfully.
</td><td>pass/fail</td></tr>
</table>

### All Product Groups
<table>
 <tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>Release blockers</td><td>Verify there are no outstanding release blocking tickets.</td><td>
  
1. Check [this](https://github.com/fleetdm/fleet/labels/~release%20blocker) filter to view all open `~release blocker` tickets.
2. If any are found raise an alarm in the `#help-engineering` and `#g-mdm` (or `#g-endpoint-ops`)  channels.
</td><td>pass/fail</td>
<tr><td>Load tests - minor releases only unless otherwise specified</td><td>Verify all load test metrics are within acceptable range on final build of RC.</td><td>
  
1. Check [this Google doc](https://docs.google.com/document/d/1V6QtFzcGDsLnn2PIvGin74DAxdAN_3likjxSssOMMQI/edit?tab=t.0#heading=h.15acjob4ji20) to review load test key metrics and checks.
2. After all expected changes have been merged to the RC branch, two load tests will need to be run - a new instance with no data, and a migrated instance.
3. For the new instance with no data, set up a load test environment using the RC branch and allow it at least 24hrs of run time.
4. For the migrated instance, set up a load test environment on the previous minor release branch. Once the environment has been set up and stabilized, follow the instructions in [Deploying code changes to fleet](https://github.com/fleetdm/fleet/blob/main/infrastructure/loadtesting/terraform/readme.md#deploying-code-changes-to-fleet) to migrate to the RC branch. Monitor the metrics post-migration to determine if any performance issues arise.
5. Record metrics in [this spreadsheet](https://docs.google.com/spreadsheets/d/1FOF0ykFVoZ7DJSTfrveip0olfyRQsY9oT1uXCCZmuKc/edit?usp=drive_link) for the two load test runs. 
</td><td>pass/fail</td></tr> 
</table>

### Notes

Issues found new to this version:

Issues found that reproduce in last stable version: 

What has not been tested:

Include any notes on whether issues should block release or not as needed:

<br>
<br>

# `fleetd` agent:

Includes updates to: 
- Orbit: True / False
- Desktop: True / False
- Chrome extension: True / False

List versions changes for any component updates below: 
<!-- Remove items without updates -->
- Orbit `v1.xx.x` > `v1.xx.x`
- Desktop `v1.xx.x` > `v1.xx.x`
- Chrome extension `v1.xx.x` > `v1.xx.x`

## Testing gates for new `fleetd` release

### Goal: Ensure new `fleetd` is tested and promoted from local > edge > stable channels

1. Build a new `fleetd` from the release candidate branch as needed for Orbit, Desktop, and Chrome Extension.

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>`fleetd` local testing</td>
<td>
1. Following [Testing TUF]([url](https://github.com/fleetdm/fleet/blob/main/tools/tuf/test/README.md)) instructions create binaries for Mac, Windows, and Ubuntu using your local TUF repository and install on macOS, Linux, and Windows hosts.<br>
</td>
<td>
1. Confirm the hosts install with the updated version and are working correctly.<br>
2. Confirm any new features and/or bug fixes associated with this release are working as intended.<br>
</td>
<td>pass/fail</td></tr>
<td>`fleetd` auto-update tests</td>
<td>
1. Conduct the [`fleetd` auto-update n+1 test]([url](https://github.com/fleetdm/fleet/blob/main/tools/tuf/test/Fleetd-auto-update-test-guide.md))<br>
2. QA certifies new release by commenting in issue.<br>
</td>
<td>
1. Agent successfully auto-updates.<br>
2. Issue is certified by QA.<br>
</td>
<td>pass/fail</td></tr>
<td>`fleetd` tests</td>
<td>
1. Set up a host in your instance to receive updates from the `edge` channels.<br>
2. Work with engineer leading the release to push changes to the `edge` channel.<br>
</td>
<td>
1. Confirm the hosts running on the edge channel receive the update and are working correctly.<br>
2. Confirm any new features and/or bug fixes associated with this release are working as intended.<br>
</td>
<td>pass/fail</td></tr></tr>

</table>

## New `fleetd` pushed to edge

### Goal: Ensure `fleetd` version pushed to edge is working with the current released version of fleet.

1. Fleet server is running the latest released version available on [Fleet Releases](https://github.com/fleetdm/fleet/releases) page.
2. Set Agent options to use edge in the Fleet server configuration. For example:<br>
 `update_channels:` <br>
  `osqueryd: edge` <br>
  `orbit: edge` <br>
  `desktop: edge` <br>
<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>

 <tr><td>Query flow</td><td>Run queries. </td><td>
1. Queries can be run manually 
</td><td>pass/fail</td></tr>

<tr><td>Host Flow</td><td>Verify a new host can be added using your own device.</td><td>
1. Hosts can enroll and report correct version of `fleetd` (orbit, osquery, desktop).<br>
2. Refetching host vitals completes and returns updated information.
</td><td>pass/fail</td></tr>

<tr><td>My device page</td><td>Verify the end user's my device page loads successfully.</td><td>
1. Clicking the Fleet desktop item, then "My device" successfully loads the my device page.<br>
2. The "My device" page is populated correctly and as expected. <br>
3. Styling and padding appears correct. 
</td><td>pass/fail</td></tr>

<tr><td>Scripts</td><td>Verify script execution</td><td>
1. Verify able to run a script on all host types from CLI.<br>
2. From Host details (macOS, Windows, & Linux) run a script that should PASS, verify.<br>
3. From Host details (macOS, Windows, & Linux) run a script that should FAIL, verify.<br>
4. Verify script results display correctly in Activity feed.
</td><td>pass/fail</td></tr>

<tr><td>Software</td><td>Verify software install / download</td><td>
1. From Host details (macOS, Windows, & Linux) run an install that should PASS, verify.<br>
2. From My Device (macOS, Windows, & Linux) software tab should have self-service items available, verify.<br>
3. Verify software installs display correctly in Activity feed.
</td><td>pass/fail</td></tr>

<tr><td>OS settings</td><td>Verify OS settings functionality</td><td>
1. Verify able to configure Disk encryption (macOS, Windows, & Linux).<br>
2. Verify host enrolled with Disk encryption enforced successfully encrypts.
</td><td>pass/fail</td></tr>

<tr><td>Packs flow</td><td>Verify management, operation, and logging of ["2017 packs"](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs).</td><td>
1. Packs successfully run on host machines after migrations <br>
2. New Packs can be created. <br>
3. Packs can be edited and deleted <br>
4. Packs results information is logged
</td><td>pass/fail</td></tr>

</table>
  
# Notes

Issues found new to this version:

Issues found that reproduce in last stable version: 

What has not been tested:


Include any notes on whether issues should block release or not as needed:
