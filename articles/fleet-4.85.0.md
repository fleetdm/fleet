# Fleet 4.85.0 | Vulnerability exposure dashboard, local admin accounts, dark mode, and more...

Fleet 4.85.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.85.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Vulnerability exposure dashboard](#vulnerability-exposure-dashboard)
- [More accurate vulnerability data](#more-accurate-vulnerability-data)
- [Pin Fleet-maintained apps to a major version](#pin-fleet-maintained-apps-to-a-major-version)
- [Create a local admin account during macOS setup](#create-a-local-admin-account-during-macos-setup)
- [Scoped API-only users](#scoped-api-only-users)
- [Dark mode](#dark-mode)


### Vulnerability exposure dashboard

Fleet now includes a vulnerability exposure report that tracks your organization's [patching progress](https://fleetdm.com/articles/how-to-use-policies-for-patch-management-in-fleet) over time. The chart covers critical vulnerabilities in major browsers, Microsoft Office, operating systems, and Adobe Reader. The report joins Fleet's growing dashboard alongside new "Hosts online" and "Hosts enrolled" reports also added in 4.85.
> The vulnerability exposure report is disabled by default (feature flag) because Fleet saw performance issues on instances with >1,000 hosts. Fleet is already using it internally, and it's a great time to start experimenting. For larger deployments, we recommend enabling it on a test fleet first before rolling out more broadly. Performance improvements are coming soon.  Note that leaving it out of a GitOps config (YAML) is a no-op for now (won't auto-enable or disable). 
GitHub issue: [#43769](https://github.com/fleetdm/fleet/issues/43769)

### More accurate vulnerability data

Fleet has migrated Red Hat Enterprise Linux (RHEL) 8 and 9 [vulnerability (CVE) scanning](https://fleetdm.com/articles/vulnerability-processing) from OVAL XML feeds to OSV JSON. This is the format Red Hat began publishing natively in November 2024. This eliminates a class of false positives: OVAL grouped CVEs by advisory and sometimes attributed them to packages that weren't actually vulnerable, while OSV maps each CVE to exact affected package versions. No Fleet configuration changes are required; the transition happens automatically on upgrade.

GitHub issue: [#40056](https://github.com/fleetdm/fleet/issues/40056)

### Pin Fleet-maintained apps to a major version

IT admins using [GitOps](https://fleetdm.com/docs/configuration/yaml-files) can now pin a [Fleet-maintained app](https://fleetdm.com/software-catalog) to a specific major version using a caret constraint (e.g. `^3`). Hosts stay patched because Fleet automatically installs updates within that major version but won't install a new major release you haven't tested or licensed. Set it once in your YAML and patching takes care of itself within the version you control. Versioning pinning in the UI is [coming soon](https://github.com/fleetdm/fleet/issues/38504).

GitHub issue: [#38988](https://github.com/fleetdm/fleet/issues/38988)

### Create a local admin account during macOS setup

During macOS [Automated Device Enrollment (ADE)](https://fleetdm.com/articles/apple-device-enrollment-program), Fleet can now create a hidden admin account. This gives IT admins a way in if hands-on access is otherwise needed. Admins can view and copy the generated password, unique per-host, from the **Host details** page. Activity is logged on account creation and password views.

GitHub issue: [#37141](https://github.com/fleetdm/fleet/issues/37141)

### Scoped API-only users

Fleet Premium now supports scoped API-only users, letting us restrict a token to a specified list of allowed API endpoints. If a token leaks, the blast radius is limited to those endpoints. Scoped API-only users can be created via the Fleet UI, [`fleetctl`](https://fleetdm.com/articles/fleetctl), or the [REST API](https://fleetdm.com/docs/rest-api/rest-api).

GitHub issue: [#38044](https://github.com/fleetdm/fleet/issues/38044)


### Dark mode

Fleet now ships with a dark theme. Now, by default, Fleet automatically follows your OS light/dark mode preference. If you want to choose, you can pick between modes on your **My account** page. Whether you're working in the dark or just prefer dark mode on principle, Fleet now looks the part.

GitHub issue: [#42977](https://github.com/fleetdm/fleet/issues/42977)

## Changes

### IT Admins

- Added a dark theme to the Fleet UI, selectable in account settings with light, dark, and system options.
- Implemented Clear Passcode feature for iOS and iPadOS.
- Added support for Fleet variables in Apple's declaration profiles (DDM).
- Added support for passing end-user authentication context to the Fleet MSI installer during Windows MDM enrollment, so end users are not prompted to authenticate twice when EUA is enabled.
- Switched to Docker as the default WiX runtime on macOS (including Apple Silicon) when generating `.msi` packages via `fleetctl package`. Wine is no longer required on macOS for the default path.
- Updated macOS 15 CIS benchmark to include v2.0.0 changes.
- Updated the macOS 14 (Sonoma) CIS policy set to benchmark v3.0.0.
- Switched Fleet-maintained apps serving location from GitHub to https://maintained-apps.fleetdm.com/manifests. If this site is inaccessible, Fleet will fall back to the previous GitHub-hosted copies of manifest files.
- Added conditional HTTP downloads using ETag headers for software in GitOps, skipping re-download when content hasn't changed.
- Added `always_download` option for software in GitOps to bypass the new conditional download feature.
- Added automatic escaping of JSON special characters in GitOps variables used in `.json` configuration profiles (Apple DDM declarations and Android profiles).
- Updated `fleetctl gitops` to process Android certificates before Android profiles.
- Made fleet name uniqueness rules consistent across the UI, API, and GitOps paths. Fleet names must now differ by more than letter case, and conflicts return a 409 error on all code paths.
- Enabled renewing and deleting AB tokens in the UI in GitOps mode.
- Changed the team's `script_execution_timeout` in agent options to default to the global agent options value when unset.
- Added ability to save policies whose SQL is flagged as a syntax error.
- Withheld Android Wi-Fi configuration profiles (`openNetworkConfiguration` with `ClientCertKeyPairAlias`) until the referenced certificate is installed or terminally failed on the device.
- Updated the host OS settings detail column to show the reason when an Android profile is pending due to a certificate dependency.
- Added "Hosts online", "Vulnerability exposure", and "Hosts enrolled" charts to the dashboard.
- Added an admin setting to control retention of vulnerability-exposure data used by the dashboard chart.
- Added new policy details page with a read-only view of policy information.
- Updated edit policy page to redirect users with read-only access to the policy details page.
- Added dedicated `/policies/:id/live` route for running policies.

### Security Engineers

- Added UI pages for creating and editing API-only users with support for fleet assignment, role selection, and API endpoint access control.
- Added new middleware (`APIOnlyEndpointCheck`) that enforces a 403 response for API-only users whose request either isn't in the API endpoint catalog or falls outside their configured per-user endpoint restrictions.
- Added `POST /users/api_only` endpoint for creating API-only users.
- Added `PATCH /users/api_only/{id}` endpoint for updating existing API-only users.
- Updated `fleetctl user create --api-only` to remove email and password field requirements.
- Added a new premium `GET /api/_version_/fleet/rest_api` endpoint that returns the contents of the embedded `api_endpoints.yml` artifact.
- Updated `GET /users/{id}` response to include the new `api_endpoints` field for API-only users.
- Added `user_api_endpoints` table to track per-user API endpoint permissions.

### Bug fixes and improvements

- Updated Go to 1.26.3.
- Improved MySQL writer performance by skipping no-op `UPDATE host_orbit_info` and `UPDATE host_disks` writes when the stored values already match the incoming ingest values from osquery, cutting these writes to near zero at steady state.
- Improved Fleet-maintained apps (FMA) sync performance by adding an index on `software.bundle_identifier` that eliminates a full table scan during the hourly sync, reducing writer CPU load on large deployments.
- Improved the performance of deleting Windows MDM configuration profiles at scale by collapsing the per-profile update loop into a single batched statement that spans multiple profiles per chunk.
- Updated copy, show, and other action buttons app-wide for a more consistent style.
- Improved button and link styling.
- Improved the OS settings modal layout.
- Improved host policy empty state.
- Updated the enrollment page enroll button to render at full screen width for larger-resolution mobile devices.
- Updated the error message returned when an invalid domain is supplied for MDM Apple CSR signing.
- Updated EULA PDF upload size check to use the default max request body size.
- Added activity when a Windows MDM wipe command fails.
- Improved documentation for MySQL read replica configuration, clarifying that all settings (including region for IAM authentication) must be explicitly set for the read replica.
- Upgraded to TypeScript 6.0 for the app frontend.
- Moved some core UI form components to TypeScript for better predictability and reliability.
- Removed the unused `windows_updates` MySQL table and ingestion code.
- Implemented the chart bounded context and schema to support charting capabilities in Fleet.
- Added `gitOpsModeEnabled` and `gitOpsModeExceptions` to the anonymous statistics payload.
- Added startup validation that panics if any route declared in `service/api_endpoints.yml` is not registered in the router.
- Stopped turning on Prometheus serving by default with a hard-coded username and password when the server is started with `--dev`.
- Fixed a Windows BitLocker encrypt/decrypt loop on machines with secondary drives using auto-unlock. Fleet now detects disk encryption using `conversion_status` (not just `protection_status`), preventing the server from repeatedly requesting encryption when the disk is already encrypted. Added `bitlocker_protection_status` tracking so the UI shows "Action required" when BitLocker protection is off instead of misleadingly showing "Verified."
- Fixed a race condition where a host could silently revert to its previous team after an admin team transfer.
- Fixed an issue where trying to wipe a device after its certificate was renewed could fail due to a missing bootstrap token. _Note: The device might still have wiped._
- Fixed a server panic (502) when an Android pubsub status report arrived for a host that had been deleted from Fleet.
- Fixed a server panic when an Apple MDM `DeviceInformation` refetch response omitted `DeviceName` or other expected fields.
- Fixed an issue where Fleet would send an `AccountConfiguration` command to iOS and iPadOS devices when end user authentication was enabled; `AccountConfiguration` is macOS-only.
- Fixed a bug where pending MDM profile rows persisted in the database after Apple or Windows MDM was turned off, causing stale profiles to reappear when MDM was re-enabled. Also fixed cleanup of pending Windows profile rows when a device unenrolls from MDM.
- Fixed a bug where custom package installers were not removed when adding an FMA for the same title via GitOps, which caused setup experience to install duplicate software.
- Fixed a bug where renaming a patch policy in a GitOps file caused it to be deleted initially.
- Fixed a bug where host environment variables in script-only packages would cause GitOps to fail.
- Fixed an issue where the DDM reconciler would not self-heal for stuck remove/pending profiles due to resend with update.
- Fixed an issue where a host DDM cleanup function was not executed for stale remove/pending profiles that weren't reported by the device.
- Fixed an issue where batch processing many DDM profile changes would result in stuck remove/pending profiles.
- Fixed an issue where sending a differently cased display name for a DDM profile via the batch endpoint would result in recreating the DDM profile and triggering a resend.
- Fixed an issue where Fleet would not remove the host OS setting entry if a `RemoveProfile` command failed with error code 89 (profile not found on device).
- Fixed an issue where adding a custom icon for a script-only package was not allowed in GitOps.
- Fixed an issue where duplicate Disk Encryption activity types showed up.
- Fixed the host details activity feed showing the previously opened host's activities by including the host ID in the activity query cache keys.
- Fixed navigation to the settings page for multi-team admin users.
- Fixed software table page number to be bookmarkable.
- Fixed an infinite page loop pagination bug on the software table page that occurred when viewing a subsequent page and then using the software filter dropdown.
- Fixed styling bugs in GitOps mode UI.
- Fixed padding between GitOps exceptions checkboxes.
- Fixed a nil pointer dereference in the contributor API spec/policies.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.85.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-05-13">
<meta name="articleTitle" value="Fleet 4.85.0 | Vulnerability exposure dashboard, local admin accounts, dark mode, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.85.0-1600x900@2x.png">
