# Fleet 4.84.0 | Python scripts, Entra for Windows, auto-rotate Recovery Lock, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/mpKDV7zpb-I?si=PG5inruNQNzHrPVw" title="0" allowfullscreen></iframe>
</div>

Fleet 4.84.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.84.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- GitOps mode exceptions
- Automatically rotate Recovery Lock passwords
- Run Python scripts on macOS & Linux
- Entra conditional access for Windows
- Remove settings from Windows when profile is deleted

### GitOps mode exceptions

Fleet now lets IT admins opt specific resources out of GitOps enforcement. When GitOps mode is enabled, admins can configure exceptions for software, labels, and enroll secrets — allowing those resources to be managed via the UI or API instead of git.

This makes it easier to ramp up with GitOps incrementally: start by managing policies and profiles in git, then add software and labels later as the team gets comfortable. Exceptions are configured per resource type and require global admin permissions. If an exception is enabled and the corresponding key is present in a YAML file, GitOps will surface a clear error during the dry run to prevent the UI-managed changes from being silently overwritten.

> **Note:**
> - After upgrading, existing Fleet instances will have the labels exception enabled automatically. This way, your next GitOps run after upgrade doesn't wipe any labels not defined in git. If your GitOps YAML files include a `labels:` key, you will encounter new errors.
> - To resolve, either remove `labels:` from your YAML files (to manage labels via the UI or API going forward) or disable the labels exception in **Settings > Integrations > Change management** (to manage labels via GitOps). If you disable the exception, make sure you move any labels managed via the UI into your YAML, otherwise your next GitOps run will wipe them out. Feel free to [reach out to Fleet](https://fleetdm.com/support) if you need a hand.

GitHub issue: [#40171](https://github.com/fleetdm/fleet/issues/40171)

### Automatically rotate Recovery Lock passwords

Fleet now automatically rotates macOS Recovery Lock passwords after an IT admin views them. Previously, Fleet escrowed a unique password per host and let IT admins rotate it on demand — but rotation was a manual step. Now, after a password is viewed, Fleet schedules an automatic rotation (1 hr after view) so passwords aren't reused.

Admins can still trigger a manual rotation at any time from the Host details page. The rotation generates an audit log entry so the action is traceable.

GitHub issue: [#41003](https://github.com/fleetdm/fleet/issues/41003)

### Run Python scripts on macOS & Linux

Fleet now supports Python scripts alongside shell (`.sh`) and PowerShell (`.ps1`) scripts. IT admins can upload `.py` files in **Controls > Scripts** and run them on macOS and Linux hosts — on demand, in bulk, or as a policy automation.

Python scripts follow the same rules as other script types: they respect Fleet's [script timeout](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout), support [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), and can be defined in GitOps.

GitHub issue: [#38793](https://github.com/fleetdm/fleet/issues/38793)

### Entra conditional access for Windows

Fleet now supports [Microsoft Entra conditional access](https://fleetdm.com/guides/entra-conditional-access-integration) for Windows hosts. IT admins can mark policies as conditional access policies targeting Windows hosts — when a host fails one of those policies, Entra blocks the end user from accessing corporate resources such as Microsoft Teams and Office.

This extends the existing macOS conditional access integration to Windows, using the same Fleet + Entra setup. To configure, head to **Integrations > Conditional access** and enable conditional access on any policy targeting Windows.

GitHub issue: [#38041](https://github.com/fleetdm/fleet/issues/38041)

### Remove settings from Windows when profile is deleted

When an IT admin deletes a Windows configuration profile in Fleet, Fleet now actively removes those settings from enrolled hosts. This ensures that hosts match the intended configuration state regardless of when they enrolled.

Previously, deleting a profile only prevented it from being applied to newly enrolled hosts. Existing hosts retained the settings silently. Now, Fleet sends a removal command so the configuration is reverted on all affected hosts.

GitHub issue: [#33418](https://github.com/fleetdm/fleet/issues/33418)

## Changes

### IT Admins
- Added support for Entra conditional access to Windows devices.
- Added ability to pin Fleet-maintained apps to a specific major version in GitOps.
- Implemented ACME for MDM protocol communication, and hardware device attestation.
- Added `GET /api/v1/fleet/hosts/{id}/reports` endpoint (also accessible as `/hosts/{id}/queries`) that lists the query reports associated with a specific host.
- Added support for `labels_include_all` conditional scoping for software installers and apps.
- Added validation for software install, uninstall, and post-install scripts.
- Added ability to specify custom patch policy query in an FMA manifest.
- Added ability to re-send Android certificates to a specific host.
- Added Reports tab to Host details page.
- Allowed specifying a Fleet-Maintained App (FMA) as a policy software automation in GitOps.
- Added support for running python scripts on macOS and Linux.
- Added automatic retry (up to 3 times) when the Android agent reports a certificate install failure.
- Added activity logging when a certificate is installed or fails to install on an Android host.
- Enabled the host activity card on the Android host details page.
- Switched Fleet-maintained apps serving location from GitHub to https://maintained-apps.fleetdm.com/manifests. **NOTE:** If you limit outbound Fleet server traffic, make sure it can access the new FMA manifests location.
- Increased automatic retry limit for failed Apple (macOS, iOS, iPadOS) configuration profiles from 1 to 3. Windows profiles remain at 1 retry.
- Added a new `disk_space` fleetd table for macOS that reports available disk space including purgeable storage, matching the value shown in Finder's "Get Info" dialog and System Settings → General → Storage.
- Added configuration profile deletion when a Windows configuration profile is deleted or a host moves teams via SyncML `<Delete>` commands, bringing Windows profile removal to parity with macOS.
- Added support for outputting VPP policy automations in `fleetctl generate-gitops`.
- Added logging of profile names alongside MDM commands installing or removing them.
- Added indication in the UI when a profile command was deferred via `NotNow` status.
- Added activity when setup experience is canceled due to software install failure.
- Added cancel activities for each VPP app install skipped due to setup experience cancellation, and switched "failed" activity to "canceled" for package-based software installs in the same situation.
- Added install failure activity when VPP installs fail due to licensing issues during setup experience.

### Security Engineers
- Added vulnerability detection for Microsoft 365 Apps and Office products on Windows.
- Added OSV data source for Ubuntu vulnerability scanning.
- Added automatic rotation of Mac recovery lock passwords 1 hour after the password is viewed via the API.
- Updated ingestion/CVE logic to support JetBrains software with 2 version numbers, like WebStorm 2025.1
- Addressed false positive vulnerabilities (CVE-2019-17201, CVE-2019-17202) reported for Admin By Request on macOS and Linux hosts. These CVEs are Windows-specific.
- Generated correct CPE from malformed ipswitch whatsup CPE, ensuring applicable CVEs are matched.
- Added software source to ecosystem matching to help prevent non-deterministic CPE selection when multiple vendors exist for the same product.

### Other improvements and bug fixes
- Upped the default limit for the software batch endpoint, from 1MiB to 25MiB.
- Added `FLEET_MDM_CERTIFICATE_PROFILES_LIMIT` server config option to throttle the number of CA certificate profile installations per reconciler cycle, preventing CA server overload in large deployments.
- Added banner to Add software page to inform users that Android web apps require Google Chrome.
- Enabled Windows MDM in `fleetctl preview` by auto-generating WSTEP certificates on startup.
- Used the same templates for `fleetctl new` and new instance initialization.
- Added "API time" to GitOps output on API errors.
- Allowed clearing Windows OS update deadline and grace period fields to remove enforcement.
- Updated ordering of setup experience software to take display names into account.
- Updated iOS/iPadOS refetch logic to slowly clear out old/stale results.
- Increased the default SSO session validity period from 5 to 15 minutes.
- Improved performance of distributed read endpoint by reducing mutex contention in shouldUpdate using sync.RWMutex instead of sync.Mutex.
- Allowed OTEL service name to be overridden with standard OTEL_SERVICE_NAME env var.
- Revised which versions Fleet tests MySQL against to remove 8.0.39 and add 8.0.42.
- Allowed typing whitespace on Settings > Integrations > SSO > End users form.
- Removed incorrect `report` key from get/create/modify API responses.
- Added `(query_id, has_data, host_id, last_fetched)` index on query_results.
- Improved database query performance for the Host Details > Reports page by adding a `has_data` virtual generated column to `query_results`.
- Made sure that fleet names are trimmed and validate to prevent whitespace-only or padded names across API, gitops, frontend, and existing data.
- Hid host details > reports in the UI from platforms that do not support scheduled reporting.
- Updated GitOps label functionality to allow omitting the `hosts:` key under a manual label to mean "preserve existing host membership", rather than removing all hosts.
- Added Flatcar Container Linux and CoreOS to the list of recognized Linux platforms, fixing host detail queries (IP address, disk space, etc.) not being sent to hosts running these distributions.
- Updated the default fleet selected when navigating to the dashboard and to controls.
- Reduced redundant database queries during policy result submission by computing flipping policies once per host check-in instead of multiple times.
- Reduced redundant database calls in the osquery distributed query results hot path by pre-loading configuration (AppConfig, HostFeatures, TeamMDMConfig, conditional access) once per request instead of once per detail query result.
- Updated UI to use new multiplatform API keys.
- Activated warnings for deprecated API parameters, API URLs, fleetctl commands and fleetctl command options.
- Updated the Request Certificate API to return the proper PEM header for PKCS #7 certificates returned by EST CAs.
- Added "Learn more" link on End User Authentication section.
- Moved Apple MDM worker to a faster cron, and started sending profiles on Post DEP enrollment job, to speed up initial macOS setup.
- Optimized `PolicyQueriesForHost` and `ListPoliciesForHost` SQL queries by replacing correlated subqueries with a single aggregated LEFT JOIN for label-based policy scoping, reducing query time by ~77% at scale.
- Improved VPP install failure messaging to explain verification timeouts in Host details and My device install details.
- Refactored large anonymous functions into named functions to improve nil-safety static analysis coverage.
- Renamed "Custom settings" to "Configuration profiles" in Fleet UI.
- Added description to UI to help users understand which fleet a policy belongs to during add/edit.
- Updated Fleet-maintained apps to overwrite software title names on sync and when adding an FMA installer.
- Improved Fleet server performance for the Windows MDM profiles summary and host OS settings filter queries by replacing correlated subqueries with a single aggregation pass.
- Improved Windows MDM server performance at scale by reducing redundant database queries during device check-ins.
- Updated go to 1.26.1
- Fixed a server panic when uploading a Windows MDM profile to a fleet on a free license.
- Fixed MSRC vulnerability scanning to differentiate between Windows Server Core and full desktop installations, preventing false positive/negative CVEs caused by non-deterministic product matching.
- Fixed GitOps policy software resolution failing when URL lookup doesn't match, by falling back to hash-based lookup.
- Fixed GitOps failing to delete a certificate authority when certificate templates still reference it in fleet configs.
- Fixed duplicate text in error message when script validation fails when adding a custom package.
- Fixed issue where the `include_available_for_install` query param wasn't being applied correctly to the `GET /api/latest/fleet/hosts/{id}/software` endpoint.
- Fixed disk encryption key modal to not show stale key when switching between hosts.
- Fixed SCIM user not associating with host when IdP username was set before the SCIM user was created.
- Fixed  Google Drive version not matching upstream.
- Fixed bug that cleared the MDM lock state if an "idle" message was received right after the lock ACK.
- Fixed team maintainers, admins, and GitOps users being unable to add certificate templates due to missing read access to certificate authorities.
- Fixed fleetd installation failure on macOS when installing it through Host details page > Software > Library as a Custom package.
- Fixed a bug where SQL queries using table aliases (e.g., `FROM mounts m`) incorrectly reported no compatible platforms.
- Fixed `fleetctl gitops` failing with "No available VPP Token" when assigning VPP apps alongside a new team.
- Fixed a bug where OS versions were not populated in vulnerability details for OS-only vulnerabilities (e.g., macOS CVEs).
- Fixed a TOCTOU-related issue when checking before deleting last admin.
- Fixed database locking issues on the policy_membership table by batching cleanup DELETE operations and moving them outside the primary GitOps apply transaction.
- Fixed success message on Android software configuration to reference software display name when applicable.
- Fixed a bug where Android host certificate template records were not cleared when a device unenrolled, causing stale certificate statuses after re-enrollment.
- Fixed a bug where the organization logo URL entered during setup was only saved for dark backgrounds and not for light backgrounds.
- Fixed an issue where setup experience items (software to install) were not enqueued for Linux distributions that did not report a "platform-like" value, e.g. Arch Linux and Omarchy.
- Fixed a bug where filtering hosts by software version for a software version not present on the selected team returned nil software instead of a lightweight report of the software.
- Fixed Fleet's usage of the incorrectly spelled 'vulnerabities' in favor of 'vulnerabilities' in MSRC bulletins.
- Fixed nondeterministic CPE matching when multiple CPE candidates share the same product name.
- Fixed a bug where Windows hosts with an empty `display_version` in the database would get 0 CVEs from MSRC vulnerability scanning.
- Fixed a bug where `fleetctl generate-gitops` failed if a Fleet-maintained app was associated to a software title with a different name (e.g. names with different versions).
- Fixed `fleetctl generate-gitops` failing to include VPP fleet assignments.
- Fixed query results table deduplicating rows when query data contains an `id` column, and fixed `id` column header and cell styling.
- Fixed missing underline on "Reports" nav item when active in top navigation.
- Fixed bug where adding a patch policy for a new installer in the UI caused gitops runs that didn't include that installer to fail.
- Fixed browser back button requiring an extra click to leave the Policies and Reports pages.
- Fixed a bug where Fleet continued to show a stale Recovery Lock password after a macOS host left MDM, by soft-deleting the stored password whenever the host leaves MDM (re-enrollment, CheckOut, admin unenroll, or a periodic sweep of hosts osquery reports as unenrolled) and hiding the password on the host details page until the host is enrolled again.
- Fixed an issue where silent migration status would persist even after re-enrolling the device normally, causing SCEP renewal to fail.
- Fixed issue where the "Change Management" form would reset when the page lost and regained focus.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.84.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-04-24">
<meta name="articleTitle" value="Fleet 4.84.0 | Python scripts, Entra for Windows, auto-rotate Recovery Lock, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.84.0-1600x900@2x.png">
