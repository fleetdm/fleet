---
name:  Release QA
about: Checklist of required tests prior to release
title: ''
labels: ''
assignees: ''

---

# Goal: easy-to-follow test steps for checking a release manually

**Fleet version** (Head to the "My account" page in the Fleet UI or run `fleetctl version`):

**Web browser** _(e.g. Chrome 88.0.4324)_: 

# Important reference data

1. [fleetctl preview setup](https://fleetdm.com/fleetctl-preview)
2. [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) 
3. premium tests require license key (needs renewal) `fleetctl preview --license-key=eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjQwOTk1MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjIyNDI2NTg2fQ.WmZ0kG4seW3IrNvULCHUPBSfFdqj38A_eiXdV_DFunMHechjHbkwtfkf1J6JQJoDyqn8raXpgbdhafDwv3rmDw`
4. premium tests require license key (active - Expires Sunday, January 1, 2023 12:00:00 AM) `fleetctl preview --license-key=eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjcyNTMxMjAwLCJzdWIiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6InByZW1pdW0iLCJpYXQiOjE2NDI1MjIxODF9.EGHQjIzM73YyMbnCruswzg360DEYCsDi9uz48YcDwQHq90BabGT5PIXRiculw79emGj5sk2aKgccTd2hU5J7Jw`


# Smoke Tests
Smoke tests are limited to core functionality and serve as a pre-release final review. If smoke tests are failing, a release cannot proceed.

## Prerequisites

1. `fleetctl preview` is set up and running the desired test version using [`--tag` parameters.](https://github.com/fleetdm/fleet/blob/main/handbook/product.md#manual-qa )
2. Unless you are explicitly testing older browser versions, browser is up to date.
3. Certificate & flagfile are in place to create new host.
4. In your browser, clear local storage using devtools.

## Instructions

<table>
<tr><th>Test name</th><th>Step instructions</th><th>Expected result</th><th>pass/fail</td></tr>
<tr><td>$Name</td><td>{what a tester should do}</td><td>{what a tester should see when they do that}</td><td>pass/fail</td></tr>
<tr><td>Update flow</td><td>

1. remove all fleet processes/agents/etc using `fleetctl preview reset` for a clean slate
1. run `fleetctl preview` with no tag for latest stable
1. create a host/query to later confirm upgrade with
1. STOP fleet-preview-server instances in containers/apps on Docker
1. run `fleetctl preview` with appropriate testing tag </td><td>All previously created hosts/queries are verified to still exist</td><td>pass/fail</td></tr>
<tr><td>Login flow</td><td>

1. navigate to the login page and attempt to login with both valid and invalid credentials to verify some combination of expected results.
2. navigate to the login page and attempt to login with both valid and invalid sso credentials to verify expected results.
</td><td>

1. text fields prompt when blank
2. correct error message is "authentication failed"
3. forget password link prompts for email
4. valid credentials result in a successful login.
5. valid sso credentials result in a successful login</td><td>pass/fail</td></tr>
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

<tr><td>MDM Smoke Tests</td><td>Verify MDM enrollments, run commands, and script execution</td><td>
  
1. Erase an ADE-eligible macOS host and verify able to complete auomated enrollment flow.
2. With Windows MDM turned On, enroll a Windows host and verify MDM is turned On for the host.
3. Verify able to run a script on all host types from CLI.
4. Verify scripts library upload/download/delete, run from Host details.
5. Verify able to run MDM commands on both macOS and Windows hosts from the CLI.
</td><td>pass/fail</td></tr>

<tr><td>Migration Test</td><td>Verify Fleet can migrate to the next version with no issues.</td><td>

Using the migration scripts located in fleet/test/upgrade/
1. Run the upgrade_test.go script using the most recent stable version of Fleet and `main`.
2. Upgrade test returns an 'OK' response.
</td><td>pass/fail</td></tr>
  
<tr><td>Release blockers</td><td>Verify there are no outstanding release blocking tickets.</td><td>
  
1. Check [this](https://github.com/fleetdm/fleet/labels/~release%20blocker) filter to view all open `~release blocker` tickets.
2. If any are found raise an alarm in the `#help-engineering` and `#help-product-design` channels.
</td><td>pass/fail</td></tr>
</table>
  
# Notes

Issues found new to this version:

Issues found that reproduce in last stable version: 

What has not been tested:

Include any notes on whether issues should block release or not as needed
