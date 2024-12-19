## Fleet 4.61.0 (Dec 17, 2024)

## Endpoint operations
- Added support to require email verification (MFA) on each login when setting up a Fleet user outside SSO.
- Extended Linux encryption key escrow support to Ubuntu 20.04.6.
- Added missing APM instrumentation for Fleet API routes.
- Improved label validation when running live queries. Previously, when passing label(s) that do not exist, the labels were ignored. Now, an error is returned indicating which labels were not found. This change affects both the API and `fleetctl query` command.

## Device management (MDM)
- Added functionality for creating an automatic install policy for Fleet-maintained apps.
- Replaced Zoom Fleet-maintained app with Zoom for IT, which does not open any windows during installation.
- Added support for the new `windows_migration_enabled` setting (can be set via `fleetctl`, the `PATCH /api/latest/fleet/config` API endpoint and the UI). Requires a premium license.
- Updated to only show the "follow instructions on My device" banner for Linux hosts whose disks are encrypted but for which Fleet hasn't escrowed a valid key.
- Added App Store app UI: Added different empty state when VPP token is not added at all vs. when it's not assigned to a team to prevent confusion.
- Allowed APNS key to be in unencrypted PKCS8 format, which may happen when migrating from another MDM.
- Allowed calling `/api/v1/fleet/software/fleet_maintained_apps` with no team ID to retrieve the full global list of maintained apps.
- Added UI changes for windows MDM page and allow for automatic migration for windows hosts.
- Bypassed the setup experience UI if there is no setup experience item to process (no software to install, no script to execute), so that releasing the device is done without going through that window.

## Vulnerability management
- Added `without_vulnerability_details` to software versions endpoint (/api/latest/fleet/software/versions) so CVE details can be truncated when on Fleet Premium.
- Fixed an issue where the github cli software name was not matching against the cpe vulnerability name.

## Bug fixes and improvements
- Updated Go version to 1.23.4.
- Update help text for policy automation Install software and run script modals.
- Updated to display Windows MDM WSTEP flags in `fleet --help`.
- Added language in email templates indicating that users should not reply to the automated emails.
- Added better information on what deleting a host does.
- Added a clearer error message when users attempt to turn MDM off on a Windows host.
- Improved side nav empty state UI under `/settings`.
- Added missing loading spinner for delete modals (delete configuration profile, delete script, delete setup script and delete software).
- Improved performance of updating the `nano_enrollments.last_seen_at` timestamp of Apple MDM devices by an order of magnitude under load.
- Improved MDM `SELECT FROM nano_enrollment_queue` MySQL query performance, including calling it on DB reader much of the time.
- Updated Inter font to latest version for woff2 files.
- Added better documentation around how the --label flag works in the fleetctl query command.
- Switched Twitter logo to X logo in Fleet-initiated automated emails.
- Removed duplicate indexes from the database schema..
- Added cleanup job to delete stuck pending Apple profiles, and requeue them.
- Exclude any custom sourced "users" from the host details "used by" display if Fleet doesn't have an email for them.
- Replaced the internal use of the deprecated `go.mozilla.org/pkcs7` package with the maintained fork `github.com/smallstep/pkcs7`.
- Switched email template font to Inter to match previous changes in the rest of the UI.
- Updated resend config profile API from `hosts/[hostid}/configuration_profiles/resend/{uuid}` to `hosts/{hostid}/configuration_profiles/{uuid}/resend`.
- Update nanomdm dependency with latest bug fixes and improvements.
- Updated documentation to include `firefox_preferences` table for Linux and Windows platforms.
- Restored the user's previous scroll, if any, when they change the filter on the host software table.
- Updated a link in the Fleet-maintained apps UI to point to the correct place.
- Removed image borders that are included in Apple's app store icons.
- Redirect when user provides an invalid URL param for fleet-maintained software id.
- Added additional statistics item for number of saved queries.
- Fixed a bug where the name of the setup experience script was not showing up in the activity for that script execution.
- Present a nicely formatted and more informative UI for log destination in two places.
- Fixed bug in `fleetdm/fleetctl` docker image where the `build` directory does not exist when generating deb/rpm packages.
- Fixed missing read permission for team maintainers and admins on Fleet maintained apps.
- Fixed a bug that would add "Fleet" to activities where it shouldn't be.
- Fixed ability to clear policy automation that empties webhook URL.
- Fixes a bug with pagination in the profiles and scripts lists.
- Fixed duplicate queries in query stats list in host details.
- Fixed zip and dmg automations showing null platform for installer
- Fixed a typo in the loading modal when adding a Fleet-maintained app.
- Fixed UI bug where "Actions" dropdown on host software page included "Install" and "Uninstall" options for software that is not able to be installed via Fleet.
- Fixed a bug where the HTTP client used for MDM APNs push notifications did not support using a configured proxy.
- Fixed potential deadlocks when deploying Apple configuration profiles.
- Fixed releasing a DEP-enrolled macOS device if mTLS is configured for `fleetd`.
- Fixed learn more about JIT provisioning link.
- Fixed an issue with the copy for the activity generated by viewing a locked macOS host's PIN.
- Fixed breaking with gitops user role running `fleetctl gitops` command when MDM is enabled.
- Fixed responsive styles for the ADM table.

## Fleet 4.60.1 (Dec 03, 2024)

### Bug fixes

- Fixed a bug that caused breaking with gitops user role running `fleetctl gitops` command when MDM was enabled.

## Fleet 4.60.0 (Nov 27, 2024)

### Endpoint operations
- Added support for labels_include_any to gitops.
- Added major improvements to keyboard accessibility throughout app (e.g. checkboxes, dropdowns, table navigation).
- Added activity item for `fleetd` enrollment with host serial and display name.
- Added capability for Fleet to serve YARA rules to agents over HTTPS authenticated via node key (requires osquery 5.14+).
- Added a query to allow users to turn on/off automations while being transparent of the current log destination.
- Updated UI to allow users to view scripts (from both the scripts page and host details page) without downloading them.
- Updated activity feed to generate an activity when activity automations are enabled, edited, or disabled.
- Cancelled pending script executions when a script is edited or deleted.

### Device management (MDM)
- Added better handling of timeout and insufficient permissions errors in NDES SCEP proxy.
- Added info banner for cloud customers to help with their windows autoenrollment setup.
- Added DB support for "include any" label profile deployment.
- Added support for "include any" label/profile relationships to the profile reconciliation machinery.
- Added `team_identifier` signature information to Apple macOS applications to the `/api/latest/fleet/hosts/:id/software` API endpoint.
- Added indicator of how fresh a software title's host and version counts are on the title's details page.
- Added UI for allowing users to install custom profiles on hosts that include any of the defined labels.
- Added UI features supporting disk encryption for Ubuntu and Fedora Linux.
- Added support for deb packages compressed with zstd.

### Vulnerability management
- Allowed skipping computationally heavy population of vulnerability details when populating host software on hosts list endpoint (`GET /api/latest/fleet/hosts`) when using Fleet Premium (`populate_software=without_vulnerability_descriptions`).

### Bug fixes and improvements
- Improved memory usage of the Fleet server when uploading a large software installer file. Note that the installer will now use (temporary) disk space and sufficient storage space is required.
- Improved performance of adding and removing profiles to large teams by an order of magnitude.
- Disabled accessibility via keyboard for forms that are disabled via a slider.
- Updated software batch endpoint status code from 200 (OK) to 202 (Accepted).
- Updated a package used for testing (msw) to improve security.
- Updated to reboot linux machine on unlock to work around GDM bug on Ubuntu 24.04.
- Updated GitOps to return an error if the deprecated `apple_bm_default_team` key is used and there are more than 1 ABM tokens in Fleet.
- Dismissed error flash on the my device page when navigating to another URL.
- Modified the Fleet setup experience feature to not run if there is no software or script configured for the setup experience.
- Set a more accurate minimum height for the Add hosts > ChromeOS > Policy for extension field, avoiding a scrollbar.
- Added UI prompt for user to reenter the password if SCEP/NDES url or username has changed.
- Updated ABM public key to download as as PEM format instead of CRT.
- Fixed issue with uploading macOS software packages that do not have a top level `Distribution.xml`, but do have a top level `PackageInfo.xml`. For example, Okta Verify.app.
- Fixed some cases where Fleet Maintained Apps generated incorrect uninstall scripts.
- Fixed a bug where a device that was removed from ABM and then added back wouldn't properly re-enroll in Fleet MDM.
- Fixed name/version parsing issue with PE (EXE) installer self-extracting archives such as Opera.
- Fixed a bug where the create and update label endpoints could return outdated information in a deployment using a mysql replica.
- Fixed the MDM configuration profiles deployment when based on excluded labels.
- Fixed gitops path resolution for installer queries and scripts to always be relative to where the query file or script is referenced. This change breaks existing YAML files that had to account for previous inconsistent behavior (e.g. installers in a subdirectory referencing scripts elsewhere).
- Fixed issue where minimum OS version enforcement was not being applied during Apple ADE if MDM IdP integration was enabled.
- Fixed a bug where users would be allowed to attempt an install of an App Store app on a host that was not MDM enrolled.

## Fleet 4.59.1 (Nov 18, 2024)

### Bug fixes

* Added `team_identifier` signature information to Apple macOS applications to the `/api/latest/fleet/hosts/:id/software` API endpoint.

## Fleet 4.59.0 (Nov 12, 2024)

### Endpoint operations
- Updated OpenTelemetry libraries to latest versions. This includes the following changes when OpenTelemetry is enabled:
  - MySQL spans outside of HTTPS transactions are now logged.
  - Renamed MySQL spans to include the query, for easier tracking/debugging.
- Added capability for fleetd to report vital errors to Fleet server, such as when Fleet Desktop is unable to start.
  
### Device management (MDM)
- Added UI for adding a setup experience script.
- Added UI for the install software setup experience.
- Added software experience software title selection API.
- Added database migrations to support Setup Experience.
- Added support to `fleetctl gitops` to specify a setup experience script to run and software to install, for a team or no team.
- Added an Orbit endpoint (`POST /orbit/setup_experience/status`) for checking the status of a macOS host's setup experience steps.
- Added service to track install status.
- Added ability to connect a SCEP NDES proxy.
- Added SCEP proxy for Windows NDES (Network Device Enrollment Service) AD CS server, which allows devices to request certificates.
- Added error message on the My Device page when MDM is off for the host.
- Added a config field to the UI for custom MDM URLs.
- Added integration to queue setup experience software installation on automatic enrollment.
- Added a validation to prevent removing a software package or a VPP app from a team if that software is selected to be installed during the setup experience.
- Updated user permissions to allow gitops users to run MDM commands.
- Updated to remove a pending MDM device if it was deleted from current ABM.
- Updated to ensure details for a software installation run are available and accurate even after the corresponding installer has been edited or deleted.
  - **NOTE:** The database migration included with this update backfills installer data into installation details based on the currently uploaded installer. If you want to backfill data from activities (which will be more comprehensive and accurate than the migration default, but may take awhile as the entire activities table will be scanned), run this database query _after_ running database migrations:
```sql
UPDATE host_software_installs i
JOIN activities a ON a.activity_type = 'installed_software'
	AND i.execution_id = a.details->>"$.install_uuid"
SET i.software_title_name = COALESCE(a.details->>"$.software_title", i.software_title_name),
	i.installer_filename = COALESCE(a.details->>"$.software_package", i.installer_filename),
	i.updated_at = i.updated_at
```
  - The above query is optional, and is unnecessary if no software installers have been edited.

### Vulnerability management
- Added filtering Software OS view to show only OSes from a particular platform (Windows, macOS, Linux, etc.)
- Fixed issue where the vulnerabilities cron failed to complete due to a large temporary table creation when calculating host issue counts.
- Fixed Debian python package false positive vulnerabilities by removing duplicate entries for Debian python packages installed by dpkg and renaming remaining pip installed packages to match OVAL definitions.

### Bug fixes and improvements
- Fixed the ADE enrollment release device processing for hosts running an old fleetd version.
- Fixed an issue with the BYOD enrollment page where it sometimes would show a 404 page.
- Fixed issue where macOS and Linux scripts failed to timeout on long running commands.
- Fixed bug in ABM renewal process that caused upload of new token to fail.
- Fixed blank install status when retrieving install details from the activity feed when the installer package has been updated or the software has since been removed from the host.
- Fixed the svg icon for Edge.
- Fixed frontend error when trying to view install details for an install with a blank status.
- Fixed loading state for the profile status aggregate UI.
- Fixed incorrect character set header on manual Mac enrollment config download.
- Fixed `fleetctl gitops` to support VPP apps, along with setting the VPP apps to install during the setup experience.
- Fixed bug where `PATCH /api/latest/fleet/config` was incorrectly clearing VPP token<->team associations.
- Fixed issue when trying to download the manual enrollment profile when device token is expired. We now show an error for this case.
- Fixed a bug where DDM declarations would remaing "pending" forever if they were deleted from Fleet before being sent to hosts.
- Fixed a bug where policy failures of a host were not being cleared in the host details page after configuring the host to not run any policies.
- Fixed iOS and iPadOS device release during the ADE enrollment flow.
- Ignored `--delete-other-teams` flag in `fleetctl gitops` command for non-Premium license users.
- Switched Nudge deadline time for OS upgrades on macOS pre-14 hosts from 04:00 UTC to 20:00 UTC.
- Added a more descriptive error message when install or uninstall details do not exist for an activity.
- Updated to allow FLEET_REDIS_ADDRESS to include a `redis://` prefix. Allowed formats are: `redis://host:port` or `host:port`.
- Documented that Microsoft enrollments have less fields filled in the `mdm_enrolled` activity due to how this MDM enrollment flow is implemented.
- Updated UI to make entire rows of the Disk encryption table clickable.
- Updated software install activities from policy automations to be authored by "Fleet", store policy ID and name on each activity.
- Updated tooltip for bootstrap package and VPP app statuses in UI.
- Added created_at/updated_at timestamps on user create endpoint.
- Updated UI notifications so that clicking in the horizontal dimension of a flash message, outside of the message itself, and always hide flash messages when changing routes.
- Filtered out VPP apps on non-MDM enrolled devices.
- Explicitly set line heights on "add profile" messages so they are consistent cross-browser.
- Deprecated the worker-based job to release macOS devices automatically after the setup experience, replace it with the fleetd-specific "/status" endpoint that is polled by the Setup Experience dialog controlled by Fleet during the setup flow.
- Improved UI feedback when user attempts and fails to reset password.

## Fleet 4.58.0 (Oct 17, 2024)

**Endpoint Operations:**

- Added builtin label for Fedora Linux.  **Warning:** Migrations will fail if a pre-existing 'Fedora Linux' label exists. To resolve, delete the existing 'Fedora Linux' label.
- Added ability to trigger script run on policy failure.
- Updated GitOps script and software installer relative paths to now always relative to the file they're in. This change breaks existing YAML files that had to account for previous inconsistent behavior (e.g. script paths declared in no-team.yml being relative to default.yaml one directory up).
- Improved performance for host details and Fleet Desktop, particularly in environments using high volumes of live queries.
- Updated activity cleanup job to remove all expired live queries to improve API performance in environment using large volumes of live queries.  To note, the cleanup cron may take longer on the first run after upgrade.
- Added an event for when a policy automation triggers a script run in the activity feed.
- Added battery status to Windows host details.

**Device Management (MDM):**

- Added the `POST /software/fleet_maintained_apps` endpoint for adding Fleet-maintained apps.
- Added the `GET /software/fleet_maintained_apps/{app_id}` endpoint to retrieve details of a Fleet-maintained app.
- Added API endpoint to list team available Fleet-maintained apps.
- Added UI for managing Fleet-maintained apps.
- Updated add software modal to be seperate pages in Fleet UI.
- Added support for uploading RPM packages.
- Updated the request timeouts for software installer edits to be the same as initial software installer uploads.
- Updated UI for software uploads to include upload progress bar.
- Improved performance of SQL queries used to determine MDM profile status for Apple hosts.

**Vulnerability Management:**

- Fixed MSRC feed pulls (for NVD release builds) in environments where GitHub access is authenticated.

**Bug fixes and improvements:**

- Added the 'Unsupported screen size' UI on the My device page.
- Removed redundant built in label filter pills.
- Updated success messages for lock, unlock, and wipe commands in the UI.
- Restricted width of policy description wrappers for better UI.
- Updated host details about section to condense information into fewer columns at smaller widths.
- Hid CVSS severity column from Fleet Free software details > vulnerabilities sections.
- Updated UI to remove leading/trailing whitespace when creating or editing team or query names.
- Added UI improvements when selecting live query targets (e.g. styling, closing behavior).
- Updated API to return 409 instead of 500 when trying to delete an installer associated with a policy automation.
- Updated battery health definitions to be defined as cycle counts greater than 1000 or max capacity falling under 80% of designed capacity for macOS and Windows.
- Added information on how battery health is defined to the UI.
- Updated UI to surface duplicate label name error to user.
- Fixed software uninstaller script for `pkg`s to only remove '.app' directories installed by the package.
- Fixed "no rows" error when adding a software installer that matches an existing title's name and source but not its bundle ID.
- Fixed an issue with the migration adding support for multiple VPP tokens that would happen if a token is removed prior to upgrading Fleet.
- Fixed UI flow for observers to easily query hosts from the host details page.
- Fixed bug with label display names always sentence casing.
- Fixed a bug where a profile wouldn't be removed from a host if it was deleted or if the host was moved to another team before the profile was installed on the host.
- Fixed a bug where removing a VPP or ABM token from a GitOps YAML file would leave the team assignments unchanged.
- Fixed host software filter bug that resets dropdown filter on table changes (pagination, order by column, etc).
- Fixed UI bug: Edit team name closes modal.
- Fixed UI so that switching vulnerability search types does not cause page re-render.
- Fixed UI policy automation truncation when selecting software to auto-install.
- Fixed UI design bug where software package file name was not displayed as expected.
- Fixed a small UI bug where a button overlapped some copy.
- Fixed software icon for chrome packages.

## Fleet 4.57.3 (Oct 11, 2024)

### Bug fixes

- Fixed Orbit configuration endpoint returning 500 for Macs running Rapid Security Response macOS releases that are enrolled in OS major version enforcement.

## Fleet 4.57.2 (Oct 03, 2024)

### Bug fixes

- Fixed software uninstaller script for `pkg`s to only remove '.app' directories installed by the package.

## Fleet 4.57.1 (Oct 01, 2024)

### Bug fixes

- Improved performance of SQL queries used to determine MDM profile status for Apple hosts.
- Ensured request timeouts for software installer edits were just as high as for initial software installer uploads.
- Fixed an issue with the migration that added support for multiple VPP tokens, which would happen if a token was removed prior to upgrading Fleet.
- Fixed a "no rows" error when adding a software installer that matched an existing title's name and source but not its bundle ID.

## Fleet 4.57.0 (Sep 23, 2024)

**Endpoint Operations**

- Added support for configuring policy installers via GitOps.
- Added support for policies in "No team" that run on hosts that belong to "No team".
- Added reserved team names: "All teams" and "No team".
- Added support the software status filter for 'No teams' on the hosts page.
- Enable 'No teams' funcitonality for the policies page and associated workflows.
- Added reset install counts and cancel pending installs/uninstalls when GitOps installer updates change package contents.
- Added support for software installer packages, self-service flag, scripts, pre-install query, and self-service availability to be edited in-place rather than deleted and re-added.

**Device Management (MDM)**

- Added feature allowing automatic installation of software on hosts that fail policies.
- Added feature for end users to enroll BYOD devices into Fleet MDM.
- Added the ability to use Fleet to uninstall packages from hosts.
- Added an endpoint for getting an OTA MDM profile for enrolling iOS and iPadOS hosts.
- Added protocol support for OTA enrollment and automatic team assignment for hosts.
- Added validation of Setup Assistant profiles on profile upload.
- Added validation to prevent installing software on a host with a pending installation.
- Allowed custom SCEP CA certificates with any kind of extendedKeyUsage attributes.
- Modified `POST /api/latest/fleet/software/batch` endpoint to be asynchronous and added a new endpoint `GET /api/latest/fleet/software/batch/{request_uuid}` to retrieve the result of the batch upload.

**Vulnerability Management**

- Fixed a false negative vulnerability for git.
- Fixed false positive vulnerabilities for minio.
- Fixed an issue where virtual box for macOS wasn't matching against the NVD product name.
- Fixed Ubuntu python package false positive vulnerabilities by removing duplicate entries for ubuntu python packages installed by dpkg and renaming remaining pip installed packages to match OVAL definitions.

**Bug fixes and improvements**

- Updated Go to go1.23.1.
- Removed validation of APNS certificate from server startup.
- Removed invalid node keys from server logs.
- Improved the UX of turning off MDM on an offline host.
- Improved clarity of GitOps VPP app ID type errors.
- Improved gitops error message about enabling windows MDM.
- Improved messaging for VPP token constraint errors.
- Improved loading state for UI tables when no data is present yet.
- Improved permissions so that hosts can no longer access installers that aren't directly assigned to them.
- Improved verification of premium license before uploading VPP tokens.
- Added "0 items" description on empty software tables for UI consistency.
- Updated the macos target minimum version tooltip.
- Fixed logic to properly catch and log APNs errors.
- Fixed UI overflow issues with OS settings table data.
- Fixed regression for checking email used to get a signed CSR.
- Fixed bugs on enrollment profiles when the organization name contains invalid XML characters.
- Fixed an issue with cron profiles delivery failing if a Windows VM is enrolled twice.
- Fixed issue where Fleet server could start when an expired ABM certificate was provided as server config.
- Fixed self-service checkbox appearing when iOS or iPadOS app is selected.

## Fleet 4.56.0 (Sep 7, 2024)

### Endpoint operations

- Added index to `query_results` DB table to speed up finding last query timestamp for a given query and host.
- Added a link in the UI to the error message when a CSR can't be downloaded due to missing private key.
- Added a disabled overlay to the Other Workflows modal on the policy page.
- Improved performance of live queries to accommodate for higher volumes when utilizing zero-trust workflows.
- Improved `fleetctl` gitops error message when trying to change team name to a team that already exists.

### Device management

- Added server support for multiple VPP tokens.
- Added new endpoints and updated existing endpoints for managing multiple Apple Business Manager tokens.
- Added support for S3 to store MDM bootstrap packages (uses the same bucket configuration as for software installers).
- Added support to UI for self service VPP software.
- Added backend and gitops support for self service VPP.
- Added ability for MDM migrations if the host is manually enrolled to a 3rd party MDM.
- Added an offline screen to the macOS MDM migration flow.
- Added new ABM page to Fleet UI.
- Added new VPP page to the fleet UI
- Added support to track the Apple Business Manager "terms expired" API error per token, as well as a global flag that gets set as soon as one token has its terms expired.
- Updated the instructions on "My device" for MDM migrations on pre-Sonoma macOS hosts.
- Updated to allow multiple teams to be assigned to the same VPP Token.
- Updated process so that deleting installed software or VPP app now makes it available for re-installation.
- Updated to enforce minimum OS version settings during Apple Automated Device Enrollment (ADE).
- Updated ABM ingestion so that deleted iOS/iPadOS host will continue to report to Fleet as long as host is in Apple Business Manager (ABM).
- Updated so that refetching an offline iOS/iPadOS host will not add new MDM commands to the queue if previous refetch has not completed yet.
- Updated UI so that downloading a software installer package now shows the browser's built-in progress bar.
- Updated relevant documentation to include references to multiple ABM and VPP tokens.
- Consolidated Automatic Enrollment and VPP settings under the MDM settings integration page.
- Cleared apps associated with a VPP token if it's moved off of a team.

### Vulnerability management

- Added ALAS bulletins as vulnerability source for Amazon Linux (instead of OVAL for Amazon Linux 2, and adds support for Amazon Linux 1, 2022, and 2023).
- Added matching rules for July and August Microsoft 365 security updates (https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates).
- Added the following filters to `/software/titles` and `/software/versions` API endpoints: `exploit: bool`, `min_cvss_score: float`, `max_cvss_score: float`.
- Updated software titles/versions tables to allow for filtering by vulnerabilities including severity and known exploit.
- Updated to use empty CVE description when the NVD CVE feed doesn't include description entries (instead of panicking).
- Updated matching software that is not installed by Fleet so that it shows up as 'Available for install' on host details page.
- Updated base images of `fleetdm/fleetctl`, `fleetdm/bomutils` and `fleetdm/wix` to fix critical vulnerabilities found by Trivy.
- Updated vulnerability scanning to use `macos` SW target for CPEs of homebrew packages.
- Updated vulnerability scanning to not ignore software with non-ASCII en dash and em dash characters.
- Updated `GET /api/v1/fleet/vulnerabilities/{cve}` endpoint to add validation of CVE format, and a 204 response. The 204 response indicates that the vulnerability is known to Fleet but not present on any hosts.
- Updated the UI to add new empty states for searching vulnerabilities: invalid CVE format searched, a known CVE serached but not present on hosts, not a known CVE searched, exploited vulnerability empty state, operating systems empty state, new icons.

### Bug fixes and improvements

- Added support for MySQL 8.4.2 LTS.
- Updated Go to go1.22.6.
- Updated Fleet server to now accept arguments via stdin. This is useful for passing secrets that you don't want to expose as env vars, in the command line, or in the config file.
- Updated text for "Turn on MDM" banners in UI.
- Updated ABM host tooltip copy on the manage host page to clarify when host vitals will be available to view.
- Updated copy on auotmatic enrollment modal on my device page.
- Updated host details activities tooltip and empty state copy to reflect recently added capabilities.
- Updated Fleet Free so users see a Premium feature message when clicking to add software.
- Updated usage reporting to report statistics on new AI features, maintenance window, and `fleetd`.
- Fixed bug where configuration profile was still showing the old label name after the name was updated.
- Fixed a bug when a cached prepared statement gets deleted in the MySQL server itself without Fleet knowing.
- Fixed a bug where the wrong API path was used to download a software installer.
- Fixed the failing_host_count so it is never 0. This count is normally updated once an hour during cleanups_then_aggregation cron job.
- Fixed CVE-2024-4030 in Vulncheck feed incorrectly targeting non-Windows hosts.
- Fixed a bug where the "Self-service" filter for the list of software and the list of host's software did not take App Store apps into account.
- Fixed a bug where the "My device" page in Fleet Desktop did not show the self-service software tab when App Store apps were available as self-install.
- Fixed a bug where a software installer (a package or a VPP app) that has been installed on a host still shows up as "Available for install" and can still be requested to be installed after the host is transferred to a different team without that installer (or after the installer is deleted).
- Fixed the "Available for install" filter in the host's software page so that installers that were requested to be installed on the host (regardless of installation status) also show up in the list.
- Fixed UI popup messages bleeding off viewport in some cases.
- Fixed an issue with the scheduling of cron jobs at startup if the job has never run, which caused it to be delayed.
- Fixed UI to display the label names in case-insensitive alphabetical order.

## Fleet 4.55.2 (Sep 05, 2024)

### Bug fixes

- Removed validation of APNS certificate from server startup. This was no longer necessary because we now allow for APNS certificates to be renewed in the UI.
- Fixed logic to properly catch and log APNs errors.

## Fleet 4.55.1 (Aug 15, 2024)

### Bug fixes

- Added a disabled overlay to the Other Workflows modal on the policy page.
- Updated text for "Turn on MDM" banners in UI.
- Fixed a bug when a cached prepared statement got deleted in the MySQL server itself without Fleet knowing.
- Continued with an empty CVE description when the NVD CVE feed didn't include description entries (instead of panicking).
- Scheduled maintenance events are now scheduled over calendar events marked "Free" (not busy) in Google Calendar.
- Fixed a bug where the wrong API path was used to download a software installer.
- Improved fleetctl gitops error message when trying to change team name to a team that already exists.
- Updated ABM (Apple Business Manager) host tooltip copy on the manage host page to clarify when host vitals will be available to view.
- Added index to query_results DB table to speed up finding the last query timestamp for a given query and host.
- Displayed the label names in case-insensitive alphabetical order in the fleet UI.

## Fleet 4.55.0 (Aug 8, 2024)

**NOTE:** Beginning with v4.55.0, Fleet no longer supports MySQL 5.7 because it has reached [end of life](https://mattermost.com/blog/mysql-5-7-reached-eol-upgrade-to-mysql-8-x-today/#:~:text=In%20October%202023%2C%20MySQL%205.7,to%20upgrade%20to%20MySQL%208.). The minimum version supported is MySQL 8.0.36.

### Endpoint Operations

- Added support for generating `fleetd` packages for Linux ARM64.
- Added new `fleetctl package` --arch flag.
- Updated `fleetctl package` command to remove the `--version` flag. The version of the package can be controlled by `--orbit-channel` flag.
- Updated maintenance window descriptions to update regularly to match the failing policy description/resolution.
- Updated maintenance windows using Google Calendar so that calendar events are now recreated within 30 seconds if deleted or moved to the past.
  - Fleet server watches for potential changes for up to 1 week after original event time. If event is moved forward more than 1 week, then after 1 week Fleet server will check for event changes once every 30 minutes.
  - **NOTE:** These near real-time updates may add additional load to the Google Calendar API, so it is recommended to use API usage alerts or other monitoring methods.

### Device Management

- Integrated [Escrow Buddy](https://github.com/macadmins/escrow-buddy) to add enforcement of FileVault during the MacOS Setup Assistant process for hosts that are 
enrolled into teams (or no team) with disk encryption turned on. Thank you [homebysix](https://github.com/homebysix) and team!
- Updated `fleetd` to use [Escrow Buddy](https://github.com/macadmins/escrow-buddy) to rotate FileVault keys. Removed or modified internal API endpoints documented in the API for contributors.
- Added OS updates support to iOS/iPadOS devices.
- Added iOS and iPadOS device details refetch triggered with the existing `POST /api/latest/fleet/hosts/:id/refetch` endpoint.
- Added iOS and iPadOS user-installed apps to Fleet.
- Added iOS and iPadOS apps to be installed using Apple's VPP (Volume Purchase Program) to Fleet.
- Added support for VPP to GitOps.
- Added the `POST /mdm/apple/vpp_token`, `DELETE /mdm/apple/vpp_token` and `GET /vpp` endpoints and related functionality.
- Added new `GET /software/app_store_apps` and `POST /software/app_store_apps` endpoints and associated functionality.
- Added the associated VPP apps to the `GET /software/titles` and `GET /software/titles/:id` endpoints.
- Added the associated VPP apps to the `GET /hosts/:id/software` and `GET /device/:token/software` endpoints.
- Added support to delete a VPP app from a team in `DELETE /software/titles/:software_title_id/available_for_install`.
- Added `exclude_software` query parameter to "Get host by identifier" API.
- Added ability to add/remove/disable apps with VPP in the Fleet UI.
- Added a warning banner to the UI if the uploaded VPP token is about to expire/has expired.
- Added UI updates for VPP feature on host software and my device pages.
- Added global activity support for VPP-related activities.
- Added UI features for managing VPP apps for iPadOS and iOS hosts.
- Updated profile activities to include iOS and iPadOS.
- Updated Fleet UI to show OS version compliance on host details page.
- Added support for "No teams" on all software pages including adding software installers.
- Added DB migration to support VPP software features.
- Added DB migration to migrate older team configurations to the new version that includes both installers and App Store apps.
- Linux lock/unlock scripts now make use of pam_nologin to keep AD users locked out.
- Installed software list now includes Linux .deb packages that are 'on hold'.
- Added a special-case to properly name the Notion .exe Windows installer the same as how it will be reported by osquery post-install.
- Increased threshold to renew Apple SCEP certificates for MDM enrollments to 180 days.

### Vulnerability Management

- Fixed CVEs identified as 'Rejected' in NVD not matching against software.
- Fixed false negative vulnerabilities with IntelliJ IDEA CE and PyCharm CE installed via Homebrew.

### Bug fixes and improvements

- Dropped support for MySQL 5.7 and raised minimum required to MySQL 8.0.36.
- Updated software pre-install to use new GitOps format for query.
- Updated UI tooltips for pending OS settings.
- Fixed a styling issue in the controls > OS settings > disk encryption table.
- Fixed a bug in `fleetctl preview` that was causing it to fail if Docker was installed without support for the deprecated `docker-compose` CLI.
- Fixed an issue where the app-wide warning banners were not showing on the initial page load.
- Fixed a bug where the hosts page would sometimes allow excess pagination.
- Fixed a bug where software install results could not be retrieved for deleted hosts in the activity feed.
- Fixed path that was incorrect for the download software installer package endpoint `GET /software/titles/:software_title_id/package`.
- Fixed a bug that set `last_enrolled_at` during orbit re-enrollment, which caused osquery enroll failures when `FLEET_OSQUERY_ENROLL_COOLDOWN` is set.
- Fixed a bug where Fleet google calendar events generated by Fleet <= 4.53.0 were not correctly processed by 4.54.0.
- Fixed a bug where software install results could not be retrieved for deleted hosts in the activity feed.
- Fixed a bug where a software installer (a package or a VPP app) that has been installed on a host still shows up as "Available for install" and can still be requested to be installed after the host is transferred to a different team without that installer (or after the installer is deleted).

## Fleet 4.54.1 (Jul 24, 2024)

### Bug fixes

- Fixed a startup bug by performing an early restart of orbit if an agent options setting has changed.
- Implemented a small refactor of orbit subsystems.
- Removed the `--version` flag from the `fleetctl package` command. The version of the package can now be controlled by the `--orbit-channel` flag.
- Fixed a bug that set `last_enrolled_at` during orbit re-enrollment, which caused osquery enroll failures when `FLEET_OSQUERY_ENROLL_COOLDOWN` is set .
- In `fleetctl package` command, removed the `--version` flag. The version of the package can be controlled by `--orbit-channel` flag.
- Fixed a bug where Fleet google calendar events generated by Fleet <= 4.53.0 were not correctly processed by 4.54.0.
- Re-enabled cached logins after windows Unlock.

## Fleet 4.54.0 (Jul 17, 2024)

### Endpoint Operations

- Updated `fleetctl gitops` to be used to rename teams.
  - **NOTE:** `fleetctl gitops` needs to have previously run with this Fleet/fleetctl version or later.
  - The team name is changed if the YAML config is applied from the same filename as before.
- Updated `fleetctl query --hosts` to work with hostnames, host UUIDs, and/or hardware serial numbers.
- Added a host's upcoming scheduled maintenance window, if any, on the host details page of the UI and in host responses from the API.
- Added support to `fleetctl debug connection` to test TLS connection with the embedded certs.pem in
  the fleetctl executable.
- Added host's display name to calendar event descriptions.
- Added .yml and .yaml file type validation and error message to `fleetctl apply`.
- Added a tooltip to truncated text and not to untruncated values.

### Device Management (MDM)

- Added iOS/iPadOS builtin manual labels. 
  - **NOTE:** Before migrating to this version, make sure to delete any labels with name "iOS" or "iPadOS".
- Added aggregation of iOS/iPadOS OS versions.
- Added change to custom profiles for iOS/iPadOS to go from 'pending' straight to 'verified' (skip 'verifying').
- Added support for renewing SCEP certificates with custom enrollment profiles.
- Added automatic install of `fleetd` when a host turns on MDM now uses the latest released `fleetd` version.
- Added support for `END_USER_EMAIL` and `FLEET_DESKTOP` parameters to Windows MSI install package.
- Added API changes to support the `labels_include_all` and `labels_exclude_any` fields (and accept the deprecated `labels` field as an alias for `labels_include_all`).
- Added `fleetctl gitops` and `fleetctl apply` support for `labels_include_all` and `labels_exclude_any` to configure a custom setting.
- Added UI for uploading custom profiles with a target of hosts that include all/exclude any selected labels.
- Added the database migrations to create the new `exclude` column for labels associated with MDM profiles (and declarations).
- Updated host script timeouts to be configurable via agent options using `script_execution_timeout`. 
- `fleetctl` now uses a polling mechanism when running `run-script` to accommodate longer script timeout values.
- Updated the profile reconciliation logic to handle the new "exclude any" labels.
- Updated so that the `fleetd` cleanup script for macOS that will return completed when run from Fleet.
- Updated so that the `fleetd` uninstall script will return completed when run from Fleet.
- Updated script run permissions -- only admins and maintainers can run arbitrary or saved scripts (not observer or observer+).
- Updated `fleetctl get mdm_commands` to return 20 rows and support `--host` `--type` filters to improve response time.
- Updated the instructions for manual MDM enrollment on the "My device" page to be clearer and align with Apple updates.
- Updated UI to allow device users to reinstall self-service software.
- Updated API to not return a 500 status code if a host sends a command response with an invalid command uuid.
- Increased the timeout of the upload software installer endpoint to 4 minutes.
- Disabled credential caching and reboot on Windows lock.

### Vulnerability Management

- Added "Vulnerable" filter to the host details software table.
- Fixed Microsoft Office June 2024 false negative vulnerabilities and added custom vulnerability matching.
- Fixed issue where some Windows applications were getting matched against Windows OS vulnerabilities.

### Bug fixes and improvements

- Updated Go version to go1.22.4.
- Updated to render only one banner on the my device page based on priority order.
- Updated software updated timestamp tooltip.
- Removed DB error message from the UI when showing a error response.
- Updated fleetctl get queries/labels/hosts descriptions.
- Reinstated ability to sort policies by passing count.
- Improved the accuracy of the heuristic used to deterimine if a host is connected to Fleet via MDM by using osquery data for hosts that didn't send a Checkout message.
- Improved the matching of `pkg` installer files to existing software.
- Improved extraction of application name from `pkg` installers.
- Clarified various help and error texts around host identifiers.
- Hid CTA on inherited queries/policies from team level users.
- Hid query delete checkboxes from team observers.
- Hid "Self-service" in Fleet Desktop and My device page if there is no self-service software available.
- Hid the host detail page's "Run script" action from Global and Team Observer/+s.
- Aligned the "View all hosts" links in the Software titles and versions tables.
- Fixed counts for hosts with with low disk space in summary page.
- Fixed allowing Observer and Observer+ roles to download software installers.
- Fixed crash in `fleetd` installer on Windows if there are registry keys with special characters on the system.
- Fixed `fleetctl debug connection` to support server TLS certificates with intermediates.
- Fixed macOS declarations being stuck in "to be removed" state indefinitely.
- Fixed link to `fleetd` uninstall instructions in "Delete device" modal.
- Fixed exporting CSVs with fields that contain commas to render properly.
- Fixed issue where the Fleet UI could not be used to renew the ABM token after the ABM user who created the token was deleted.
- Fixed styling issues with the target inputs loading spinner on the run live query/policy page.
- Fixed an issue where special characters in HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall breaks the "installer_utils.ps1 -uninstallOrbit" step in the Windows MSI installer.
- Fixed a bug causing "No Team" OS versions to display the wrong number.
- Fixed various UI capitalizations.
- Fixed UI issue where "Script is already running" tooltip incorrectly displayed when the script is not running.
- Fixed the script details modal's error message on script timeout to reflect the newly dynamic script timeout limit, if hit.
- Fixed a discrepancy in the spacing between DataSet labels and values on Firefox relative to other browsers.
- Fixed bug that set `Added to Fleet` to `Never` after macOS hosts re-enrolled to Fleet via MDM. 

## Fleet 4.53.1 (Jul 01, 2024)

### Bug fixes

- Updated fleetctl get queries/labels/hosts descriptions.
- Fixed exporting CSVs with fields that contain commas to render properly.
- Fixed link to fleetd uninstall instructions in "Delete device" modal.
- Rendered only one banner on the my device page based on priority order.
- Hidden query delete checkboxes from team observers.
- Fixed issue where the Fleet UI could not be used to renew the ABM token after the ABM user who created the token was deleted.
- Fixed an issue where special characters in HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall broke the "installer_utils.ps1 -uninstallOrbit" step in the Windows MSI installer.
- Fixed counts for hosts with low disk space in summary page.
- Fleet UI fixes: Hide CTA on inherited queries/policies from team level users.
- Updated software updated timestamp tooltip.
- Fixed issue where some Windows applications were getting matched against Windows OS vulnerabilities.
- Fixed crash in `fleetd` installer on Windows if there are registry keys with special characters on the system.
- Fixed UI capitalizations.

## Fleet 4.53.0 (Jun 25, 2024)

### Endpoint Operations

- Enabled `fleetctl gitops` to create teams with no enroll secrets, or clear enroll secrets for an existing team.
- Added support for upgrades to `fleetd` RPMs packages.
- Changed `activities.created_at` timestamp precision to microseconds.
- Added character validation to /api/fleet/orbit/device_token endpoint.
- Cleaned up count rendering fixing clientside flashing counts.
- Improved performance by removing unnecessary database query that listed host software during
  initial page load of the "My device" page.
- Made the rendering of empty text cell values consistent. Also render the '0' value as a number instead of the default value.
- Added a server setting to configure the query report max size.
- Fixed a bug where scrollbars were always present on modal backgrounds.
- Fixed bug in `fleetctl preview` caused by creating enroll secrets.

### Device Management (MDM)

- Extended the timeout for the endpoint to upload a software installer.
- Improved the logic used by Fleet to detect if a host is currently MDM-managed.
- Added S3 config variables with a `carves_` and `software_installers` prefix.
- Fixed bug where MDM migration failed when attempting to renew enrollment profiles on macOS Sonoma devices.
- Fixed issue where Windows-specific error message was displayed when failing to parse macOS configuration profiles.
- Fixed a bug where MDM migration failed when attempting to renew enrollment profiles on macOS Sonoma devices.
- Fixed a server panic when sending a request to `/mdm/apple/mdm` without certificate headers.
- Fixed issue where profiles larger than 65KB were being truncated when stored on MySQL 8.
- Fixed a bug that prevented unused script contents to be periodically cleaned up from the database.
- Fixed UI bug where error detail was overflowing the table in "OS settings" modal in "My device"
  page UI.
- Fixed a bug where the software installer exists in the database but the installer does not exist
  in the storage.
- Added a "soft-delete" approach when deleting a host so that its script execution details are still
  available for the activities feed.
- Fixed UI bug where Zoom icon was displayed for ZoomInfo.
- Fixed issue with backwards compatibility with the deprecated `FLEET_S3_*` environment variables.
- Fixed a code linter issue where a slice was created non-empty and appended-to, instead of empty with the required capacity.

### Vulnerability Management

- Added vulnerabilities matching for applications that include an OS scope.
- Added vulnerability detection in NVD for custom ubuntu kernels.
- Removed duplicate `os_versions` results in /api/latest/fleet/vulnerabilities/:cve endpoint.
- Removed vscode false positive vulnerabilities.
- Clarified Fleet uses CVSS base score version 3.x.

## Fleet 4.52.0 (Jun 20, 2024)

### Bug fixes

* Fixed an issue where profiles larger than 65KB were being truncated when stored on MySQL 8.
* Fixed activity without public IP to be human readable.
* Made the rendering of empty text cell values consistent. Also rendered the '0' value as a number instead of the default value `---`.
* Fixed bug in `fleetctl preview` caused by creating enroll secrets.
* Disabled AI features on non-new installations upgrading from < 4.50.X to >= 4.51.X.
* Fixed various icon misalignments on the dashboard page.
* Used a "soft-delete" approach when deleting a host so that its script execution details are still available for the activities feed.
* Fixed UI bug where error detail was overflowing the table in "OS settings" modal in "My device" page UI.
* Fixed bug where MDM migration failed when attempting to renew enrollment profiles on macOS Sonoma devices.
* Fixed queries with dot notation in the column name to show results.
* `/api/latest/fleet/hosts/:id/lock` returns `unlock_pin` for Apple hosts when query parameter `view_pin=true` is set. UI no longer uses unlock pending state for Apple hosts.
* Improved the logic used by Fleet to detect if a host is currently MDM-managed.
* Fixed issue where the MDM ingestion flow would fail if an invalid enrollment reference was passed.
* Removed vscode false positive vulnerabilities.
* Fixed a code linter issue where a slice was created non-empty and appended-to, instead of empty with the required capacity.
* Fixed UI bug where Zoom icon was displayed for ZoomInfo.
* Error with 404 when the user attempts to delete team policies for a non-existent team.
* Fixed the Linux unlock script to support passwordless users.
* Fixed an issue with the Windows-specific `windows-remove-fleetd.ps1` script provided in the Fleet repository where running the script did remove `fleetd` but made it impossible to reinstall the agent.
* Fixed host details page and device details page not showing the latest software. Added `exclude_software` query parameter to the `/api/latest/fleet/hosts/:id` endpoint to exclude software from the response.
* Fixed the `/mdm/apple/mdm` endpoint so that it returns status code 408 (request timeout) instead of 500 (internal server error) when encountering a timeout reading the request body.
* Extended the timeout for the endpoint to upload a software installer (`POST /fleet/software/package`), and improved handling of the maximum size.
* Fixed issue where Windows-specific error message was displayed when failing to parse macOS configuration profiles.
* Fixed a panic (API returning code 500) when the software installer exists in the database but the installer does not exist in the storage.

## Fleet 4.51.1 (Jun 11, 2024)

### Bug fixes

* Added S3 config variables with a `carves_` and `software_installers` prefix, which were used to configure buckets for those features. The existing non-prefixed variables were kept for backwards compatibility.
* Fixed a bug that prevented unused script contents to be periodically cleaned up from the database.

## Fleet 4.51.0 (Jun 10, 2024)

### Endpoint Operations
- Added support for environment variables in configuration profiles for GitOps.
- `fleetctl gitops --dry-run` now errors on duplicate (or conflicting) global/team enroll secrets.
- Added `activities_webhook` configuration option to allow for a webhook to be called when an activity is recorded. This can be used to send activity data to external services. If the webhook response is a 429 error code, the webhook retries for up to 30 minutes.
- Added Tuxedo OS to the Linux distribution platform list.

### Device Management (MDM)
- **NOTE:** Added new required Fleet server config environment variable when MDM is enabled,
  `FLEET_SERVER_PRIVATE_KEY`. This variable contains the private key used to encrypt the MDM
  certificates and keys stored in Fleet. Learm more at
  https://fleetdm.com/learn-more-about/fleet-server-private-key.
- Added MDM support for iPhone/iPad.
- Added software self-service support. 
- Added query parameter `self_service` to filter the list of software titles and the list of a host's software so that only those available to install via self-service are returned.
- Added the device-authenticated endpoint `POST /device/{token}/software/install/{software_title_id}` to self-install software.
- Added new endpoints to configure ABM keypairs and tokens.
- Added `GET /fleet/mdm/apple/request_csr` endpoint, which returns the signed APNS CSR needed to activate Apple MDM.
- Added the ability to automatically log off and lock out `Administrator` users on Windows hosts.
- Added clearer error messages when attempting to set up Apple MDM without a server private key configured.
- Added UI for the global and host activities for self-service software installation.
- Updated UI to support new workflows for macOS MDM setup and credentials.
- Updated UI to support software self-service features.
- Updated UI controls page language and hid CTA button for users without access to turn on MDM.

### Vulnerability Management
- Updated the CIS policies for Windows 11 Enterprise from v2.0.0 (03-07-2023) to v3.0.0 (02-22-2024).
- Fleet now detects Ubuntu kernel vulnerabilities from the Canonical OVAL feed.
- Fleet now detects and reports vulnerabilities on Firefox ESR editions on macOS.

### Bug fixes and improvements
- Fixed a bug that might prevent enqueuing commands to renew SCEP certificates if the host was enrolled more than once.
- Prevented the `host_id`s field from being returned from the list labels endpoint.
- Improved software ingestion performance by deduplicating incoming software.
- Placed all form field label tooltips on top.
- Fixed a number of related issues with the filtering and sorting of the queries table.
- Added various optimizations to the rendering of the queries table.
- Fixed host query page styling bugs.
- Fixed a UI bug where "Wipe" action was not being hidden from observers.
- Fixed UI bug for builtin label names for selecting targets.
- Removed references to Administrator accounts in the comments of the Windows lock script.

## Fleet 4.50.2 (May 31, 2024)

### Bug fixes

* Fixed a critical bug where S3 operation were not possible on a different AWS account.

## Fleet 4.50.1 (May 29, 2024)

### Bug fixes

* Fixed a bug that might prevent enqueing commands to renew SCEP certificates if the host was enrolled more than once.
* Fixed a bug by preventing the `host_id`s field from being returned from the list labels endpoint.
* Fixed a number of related issues with the filtering and sorting of the queries table.
* Added various optimizations to the rendering of the queries table.
* Fixed a bug where Bulk Host Delete and Transfer now support status and labelID filters together.
* Added the ability to automatically log off and lock out `Administrator` users on Windows hosts.
* Removed references to Administrator accounts in the comments of the Windows lock script.
## Fleet 4.50.0 (May 22, 2024)

### Endpoint Operations

- Added optional AI-generated policy descriptions and remediations. 
- Added flag to enable deletion of old activities and associated data in cleanup cron job.
- Added support for escaping `$` (with `\`) in gitops yaml files.
- Optimized policy_stats updates to not lock the policy_membership table.
- Optimized the hourly host_software count query to reduce individual query runtime.
- Updated built-in labels to support being applied via `fleetctl apply`.

### Device Management (MDM)

- Added endpoints to upload, delete, and download software installers.
- Added ability to upload software from the UI.
- Added functionality to filter hosts by software installer status.
- Added support to the global activity feed for "Added software" and "Deleted software" actions.
- Added the `POST /api/fleet/orbit/software_install/result` endpoint for fleetd to send results for a software installation attempt.
- Added the `GET /api/v1/fleet/hosts/{id}/software` endpoint to list the installed software for the host.
- Added support for uploading and running zsh scripts on macOS and Linux hosts.
- Added the `cron` job to periodically remove unused software installers from the store.
- Added a new command `fleetctl api` to easily use fleetctl to hit any REST endpoint via the CLI.
- Added support to extract package name and version from software installers.
- Added the uninstalled but available software installers to the response payload of the "List software titles" endpoint.
- Updated MySQL host_operating_system insert statement to reduce table lock time.
- Updated software page to support new add software feature.
- Updated fleetctl to print team id as part of the `fleetctl get teams` command.
- Implemented an S3-based and local filesystem-based storage abstraction for software installers.

### Vulnerability Management

- Added OVAL vulnerability scanning support on Ubuntu 22.10, 23.04, 23.10, and 24.04.

### Bug fixes and improvements

- Fixed ingestion of private IPv6 address from agent.
- Fixed a bug where a singular software version in the Software table generated a tooltip unnecessarily.
- Fixed bug where updating user via `/api/v1/fleet/users/:id` endpoint sometimes did not update activity feed.
- Fixed bug where hosts query results were not cleared after transferring the host to other teams.
- Fixed a bug where the returned `count` field included hosts that the user did not have permission to see.
- Fixed issue where resolved_in_version was not returning if the version number differed by a 4th part.
- Fixed MySQL sort buffer overflow when fetching activities.
- Fixed a bug with users not being collected on Linux devices.
- Fixed typo in Powershell scripts for installing Windows software.
- Fixed an issue with software severity column display in Fleet UI.
- Fixed the icon on Software OS table to show a Linux icon for Linux operating systems.
- Fixed missing tooltips in disabled "Calendar events" manage automations dropdown option.
- Updated switched accordion text.
- Updated sort the host details page queries table case-insensitively.
- Added support for ExternalId in STS Assume Role APIs.

## Fleet 4.49.4 (May 20, 2024)

### Bug fixes

* Fixed an issue with SCEP renewals that could prevent commands to renew from being enqueued.

## Fleet 4.49.3 (May 06, 2024)

### Bug fixes

* Improved Windows OS version reporting.
* Fixed a bug where when updating a policy's 'platform' field, the aggregated policy stats were not cleared.
* Improved URL and email validation in the UI.

## Fleet 4.49.2 (Apr 30, 2024)

### Bug fixes

* Restored missing tooltips when hovering over the disabled "Calendar events" manage automations dropdown option.
* Fixed an issue on Windows hosts enrolled in MDM via Azure AD where the command to install Fleetd on the device was sent repeatedly, even though `fleetd` had been properly installed.
* Improved handling of different scenarios and edge cases when hosts turned on/off MDM.
* Fixed issue with uploading of some signed Apple mobileconfig profiles.
* Added an informative flash message when the user tries to save a query with invalid platform(s).
* Fixed bug where Linux host wipe would repeat if the host got re-enrolled.

## Fleet 4.49.1 (Apr 26, 2024)

### Bug fixes

* Fixed a bug that prevented the Fleet server from starting if Windows MDM was configured but Apple MDM wasn't.

## Fleet 4.49.0 (Apr 24, 2024)

### Endpoint operations

- Added integration with Google Calendar for policy compliance events.
- Added new API endpoints to add/remove manual labels to/from a host.
- Updated the `POST /api/v1/fleet/labels` and `PATCH /api/v1/fleet/labels/{id}` endpoints to support creation and update of manual labels.
- Implemented changes in `fleetctl gitops` for batch processing queries and policies.
- Enabled setting host status webhook at the team level via REST API and fleetctl apply/gitops.

### Device management (MDM)

- Added API functionality for creating DDM declarations, both individually and as a batch.
- Added creation or update of macOS DDM profile to enforce OS Updates settings whenever the settings are changed.
- Updated `fleetctl run-script` to include new `--team` and `--script-name` flags.
- Displayed disk encryption status in macOS as "verifying" while verifying the escrowed key.
- Added the `enable_release_device_manually` configuration setting for teams and no team, which controls the automatic release of a macOS DEP-enrolled device.

### Vulnerability management

- Ignored Valve Corporation's Steam client's vulnerabilities on Windows and macOS due to retrieval challenges of the true version.
- Updated the GET fleet/os_versions and GET fleet/os_versions/[id] to restrict team users from accessing os versions on hosts from other teams.

### Bug fixes and improvements

- Upgraded Golang version to 1.21.7.
- Added a minimum supported node version in the `package.json`.
- Made block_id mismatch errors more informative as 400s instead of 500s.
- Added Windows MDM support to the `osquery-perf` host-simulation command.
- Updated calendar events automations to not show error validation on enabling the feature.
- Migrated MDM-related endpoints to new paths while maintaining support for old endpoints indefinitely.
- Added a missing database index to the MDM Windows enrollments table to improve performance at scale.
- Added cross-platform check for duplicate MDM profiles names in batch set MDM profiles API.
- Fixed a bug where Microsoft Edge was not reporting vulnerabilities.
- Fixed an issue with the `20240327115617_CreateTableNanoDDMRequests` database migration.
- Fixed the error message to indicate if a conflict on uploading an Apple profile was caused by the profile's name or its identifier.
- Fixed license checks to allow migration and restoring DEP devices during trial.
- Fixed a 500 error in MySQL 8 and when DB user has insufficient privileges for `fleetctl debug db-locks` and `fleetctl debug db-innodb-status`.
- Fixed a bug where values not derived from "actual" fleetd-chrome tables were not being displayed correctly.
- Fixed a bug where values were not being rendered in host-specific query reports.
- Fixed an issue with automatic release of the device after setup when a DDM profile is pending.
- Fixed UI issues: alignment bugs, padding around empty states, tooltip rendering, and incorrect rendering of the global Host status expiry settings page.
- Fixed a bug where `null` or excluded `smtp_settings` caused a UI 500 error.
- Fixed an issue where a bad request response from a 3rd party MDM solution would result in a 500 error in Fleet during MDM migration.
- Fixed a bug where updating policy name could result in multiple policies with the same name in a team.
- Fixed potential server panic when events are created with calendar integration, but then global calendar integration is disabled.
- Fixed fleetctl gitops dry-run validation issues when enabling calendar integration for the first time.
- Fixed a bug where all Windows MDM enrollments were detected as automatic.

## Fleet 4.48.3 (Apr 16, 2024)

### Bug fixes

* Updated calendar webhook to retry if it receives response 429 "Too Many Requests". Webhook request will retry for 30 minutes with a 1 minute max delay between retries.
* Updated label endpoints and UI to prevent creating, updating, or deleting built-in labels.
* Fixed edge cases of team ID being lost in various flows.
* Fixed queries to correctly parse params for `GET` ...`policies/count`, `GET` ...`teams/:id/policies/count`, and `GET` ...`vulnerabilities`.
* Fixed 'GET` ...`labels` to return `400` when the non-supported `query` url param was included in the request. Previous behavior was to silently ignore that param and return `200`.
* Casted windows exit codes to signed integers to match windows interpreter.
* Fixed a bug where some scripts got stuck in "upcoming" activity permanently.
* Fixed a bug where the translate API returned "forbidden" instead of "bad request" for an empty JSON body.
* Fixed an uncaught bug where "forbidden" would be returned for invalid payload type, which should also be a bad request.
* Fixed an issue where applying Windows MDM profiles using `fleetctl apply` would cause Fleet to overwrite the reserved profile used to manage Windows OS updates.
* Fixed a bug where we were not ignoreing leading and trailing whitespace when filtering Fleet entities by name.
* Fixed a bug where query retrieving bitlocker info from windows server wouldn't return.
* Fixed MDM migration starting when the device didn't have the right ADE JSON profile already assigned.

## Fleet 4.48.2 (Apr 09, 2024)

### Bug fixes

* Fixed an issue with the `20240327115617_CreateTableNanoDDMRequests` database migration where it could fail if the database did not default to the `utf8mb4_unicode_ci` collation.
* Fixed an issue with automatic release of the device after setup when a DDM profile is pending.

## Fleet 4.48.1 (Apr 08, 2024)

### Bug fixes

- Made block_id mismatch errors more informative as 400s instead of 500s
- Fixed a bug where values were not being rendered in host-specific query reports
- Fixed potential server panic when events are created with calendar integration, but then global calendar integration is disabled

## Fleet 4.48.0 (Apr 03, 2024)

### Endpoint operations
- Added integration with Google Calendar.
  * Fleet admins can enable Google Calendar integration by using a Google service account with domain-wide delegation.
  * Calendar integration is enabled at the team level for specific team policies.
  * If the policy is failing, a calendar event will be put on the host user's calendar for the 3rd Tuesday of the month.
  * During the event, Fleet will fire a webhook. IT admins should use this webhook to trigger a script or MDM command that will remediate the issue.
- Reduced the number of 'Deadlock found' errors seen by the server when multiple hosts share the same UUID.
- Removed outdated tooltips from UI.
- Added hover states to clickable elements.
- Added cross-platform check for duplicate MDM profiles names in batch set MDM profiles API.

### Device management (MDM)
- Added Windows MDM support to the `osquery-perf` host-simulation command.
- Added a missing database index to the MDM Windows enrollments table that will improve performance at scale.
- Migrate MDM-related endpoints to new paths, deprecating (but still supporting indefinitely) the old endpoints.
- Adds API functionality for creating DDM declarations, both individually and as a batch.
- Added DDM activities to the fleet UI.
- Added the `enable_release_device_manually` configuration setting for a team and no team. **Note** that the macOS automatic enrollment profile cannot set the `await_device_configured` option anymore, this setting is controlled by Fleet via the new `enable_release_device_manually` option.
- Automatically release a macOS DEP-enrolled device after enrollment commands and profiles have been delivered, unless `enable_release_device_manually` is set to `true`.

### Vulnerability management
- Added Visual Studio extensions to Fleet's software inventory.

### Bug fixes
- Fixed a bug where valid MDM enrollments would show up as unmanaged (EnrollmentState 3).
- Fixed flash message from closing when a modal closes.
- Fixed a bug where OS version information would not get detected on Windows Server 2019.
- Fixed issue where getting host details failed when attempting to read the host's bitlocker status from the datastore.
- Fixed false negative vulnerabilities on macOS Homebrew python packages.
- Fixed styling of live query disabled warning.
- Fixed issue where Windows MDM profile processing was skipping `<Add>` commands.
- Fixed UI's ability to bulk delete hosts when "All teams" is selected.
- Fixed error state rendering on the global Host status expiry settings page, fix error state alignment for tooltip-wrapper field labels across organization settings.
- Fixed `GET fleet/os_versions` and `GET fleet/os_versions/[id]` so team users no longer have access to os versions on hosts from other teams.
- `fleetctl gitops` now batch processes queries and policies.
- Fixed UI bug to render the query platform correctly for queries imported from the standard query library.
- Fixed issue where microsoft edge was not reporting vulnerabilities.
- Fixed a bug where all Windows MDM enrollments were detected as automatic.
- Fixed a bug where `null` or excluded `smtp_settings` caused a UI 500.
- Fixed query reports so they reset when there is a change to the selected platform or selected minimum osquery version.
- Fixed live query sort of sql result sort for both string and numerical columns.

## Fleet 4.47.3 (Mar 26, 2024)

### Bug fixes

* Fixed a bug where valid Windows MDM enrollments would show up as unmanaged (EnrollmentState 3).

## Fleet 4.47.2 (Mar 22, 2024)

### Bug fixes

* Fixed false negative vulnerabilities on macOS Homebrew Python packages.
* Fixed policies to check "disable guest user".
* Resolved the issue where Microsoft Edge was not reporting vulnerabilities.

## Fleet 4.47.1 (Mar 18, 2024)

### Bug fixes

* Removed outdated tooltips from UI.
* Fixed an issue with Windows MDM profile processing where `<Add>` commands were being skipped.
* Team users no longer have access to OS versions on hosts from other teams for GET fleet/os_versions and GET fleet/os_versions/[id].
* Reduced the number of 'Deadlock found' errors seen by the server when multiple hosts share the same UUID.

## Fleet 4.47.0 (Mar 11, 2024)

### Endpoint operations
- Implemented UI for team-specific host status webhooks.
- Added Unicode and emoji support for policy and team names.
- Allowed gitops user to access specific endpoints.
- Enabled setting host status webhook at the team level via REST API and fleetctl.
- GET /hosts API endpoint now populates policies with `populate_policies=true` query parameter.
- Supported custom options set via CLI in the UI for host status webhook settings.
- Surfaced VS code extensions in the software inventory.
- Added a "No team" team option when running live queries from the UI.
- Fixed tranferring hosts between teams across multiple pages.
- Fixed policy deletion not updating policy count.
- Fixed RuntimeError in fleetd-chrome and buggy filters for exporting hosts.

### Device management (MDM)
- Added wipe command to fleetctl and the `POST /api/v1/fleet/hosts/:id/wipe` Fleet Premium API endpoint.
- Updated `fleetctl run-script` to include new flags and `POST /scripts/run/sync` API to receive new parameters.
- Enabled usage of `<Add>` nodes in Windows MDM profiles.
- Added backend functionality for the new way of storing script contents and updated the script character limit.
- Updated the database schema to support the increase in script size.
- Prevented running cleanup tasks and re-enqueuing commands for hosts on SCEP renewals.
- Improved osquery queries for MDM detection.
- Prevented redundant ADE profile assignment.
- Updated fleetctl gitops, default MDM configs were set to default values when not defined.
- Displayed disk encryption status in macOS as "verifying."
- Allowed GitOps user to access MDM hosts and profiles endpoints.
- Added UI for wiping a host with Fleet MDM.
- Rolled up MDM solutions by name on the dashboard MDM card.
- Added functionality to surface MDM devices where DEP assignment failed.
- Fixed MDM profile installation error visibility.
- Fixed Windows MDM profile command "Type" column display.
- Fixed an issue with macOS ADE enrollments getting a "method not allowed" error.
- Fixed Munki issues truncated tooltip bug.
- Fixed a bug causing Windows hosts to appear when filtering by bootstrap package status.

### Vulnerability management
- Reduced vulnerability processing time by optimizing the vulnerability dictionary grouping.
- Fixed an issue with `mdm.enable_disk_encryption` JSON null values causing issues.
- Fixed vulnerability processing for non-ASCII software names.

### Bug fixes and improvements
- Upgraded Golang version to 1.21.7.
- Updated page descriptions and fixed alignment of critical policy checkboxes.
- Adjusted font size for tooltips in the settings page to follow design guidelines.
- Fixed a bug where the "Done" button on the add hosts modal could be covered.
- Fixed UI styling and alignment issues across various pages and modals.
- Fixed the position of live query/policy host search icon and UI loading states.
- Fixed issues with how errors were captured in Sentry for improved precision and coverage.

## Fleet 4.46.2 (Mar 4, 2024)

### Bug fixes

* Fixed a bug where the pencil icons next to the edit query name and description fields were inconsistently spaced.
* Fixed an issue with `mdm.enable_disk_encryption` where a `null` JSON value caused issues with MDM profiles in the `PATCH /api/v1/fleet/config` endpoint.
* Displayed disk encryption status in macOS as "verifying" while Fleet verified if the escrowed key could be decrypted.
* Fixed UI styling of loading state for automatic enrollment settings page.

## Fleet 4.46.1 (Feb 27, 2024)

### Bug fixes

* Fixed a bug in running queries via API.
	- Query campaign not clearing from Redis after timeout
* Added logging when a Redis connection is blocked for a long time waiting for live query results.
* Added support for the `redis.conn_wait_timeout` configuration setting for Redis standalone (it was previously only supported on Redis cluster).
* Added Redis cleanup of inactive queries in a cron job, so temporary Redis failures to stop a live query doesn't leave such queries around for a long time.
* Fixed orphaned live queries in Redis when client terminates connection 
	- `POST /api/latest/fleet/queries/{id}/run`
	- `GET /api/latest/fleet/queries/run`
	- `POST /api/latest/fleet/hosts/identifier/{identifier}/query` 
	- `POST /api/latest/fleet/hosts/{id}/query`
* Added --server_frequent_cleanups_enabled (FLEET_SERVER_FREQUENT_CLEANUPS_ENABLED) flag to enable cron job to clean up stale data running every 15 minutes. Currently disabled by default.

## Fleet 4.46.0 (Feb 26, 2024)

### Changes

* Fixed issues with how errors were captured in Sentry:
        - The stack trace is now more precise.
        - More error paths were captured in Sentry.
        - **Note: Many more entries could be generated in Sentry compared to earlier Fleet versions. Sentry capacity should be planned accordingly.**
- User settings/profile page officially renamed to account page
- UI Edit team more properly labeled as rename team
- Fixed issue where the "Type" column was empty for Windows MDM profile commands when running `fleetctl get mdm-commands` and `fleetctl get mdm-command-results`.
- Upgraded Golang version to 1.21.7
- Updated UI's empty policy states
* Automatically renewed macOS identity certificates for devices 30 days prior to their expiration.
* Fixed bug where updating policy name could result in multiple policies with the same name in a team.
  - This bug was introduced in Fleet v4.44.1. Any duplicate policy names in the same team were renamed by adding a number to the end of the policy name.
- Fixed an issue where some MDM profile installation errors would not be shown in Fleet.
- Deleting a policy updated the policy count
- Moved show query button to show in report page even with no results
- Updated page description styling
- Fixed UI loading state for software versions and OS for the initial request.

## Fleet 4.45.1 (Feb 23, 2024)

### Bug fixes

* Fixed a bug that caused macOS ADE enrollments gated behind SSO to get a "method not allowed" error.
* Fixed a bug where the "Done" button on the add hosts modal for plain osquery could be covered.

## Fleet 4.45.0 (Feb 20, 2024)

### Changes

* **Endpoint operations**:
  - Added two new API endpoints for running provided live query SQL on a single host.
  - Added `fleetctl gitops` command for GitOps workflow synchronization.
  - Added capabilities to the `gitops` role to support reading queries/policies and writing scripts.
  - Updated policy names to be unique per team.
  - Updated fleetd-chrome to use the latest wa-sqlite v0.9.11.
  - Updated "Add hosts" modal UI to dynamically include the `--enable-scripts` flag.
  - Added count of upcoming activities to host vitals UI.
  - Updated UI to include upcoming activity counts in host vitals.
  - Updated 405 response for `POST` requests on the root path to highlight misconfigured osquery instances.

* **Device management (MDM)**:
  - Added MDM command payloads to the response of `GET /api/_version_/fleet/mdm/commandresults`.
  - Changed several MDM-related endpoints to be platform-agnostic.
  - Added script capabilities to UI for Linux hosts.
  - Added UI for locking and unlocking hosts managed by Fleet MDM.
  - Added `fleetctl mdm lock` and `fleetctl mdm unlock` commands.
  - Added validation to reject script enqueue requests for hosts without fleetd.
  - Added the `host_mdm_actions` DB table for MDM lock and wipe functionality.
  - Updated backend MDM migration flow and added logging.
  - Updated UI text for disk encryption to reflect cross-platform functionality.
  - Renamed and updated fields in MDM configuration profiles for clarity.
  - Improved validation of Windows profiles to prevent delivery errors.
  - Improved Windows MDM profile error tooltip messages.
  - Fixed MDM unlock flow and updated lock/unlock functionality for Windows and Linux.
  - Fixed a bug that would cause OS Settings verification to fail with MySQL's `only_full_group_by` mode enabled.

* **Vulnerability management**:
  - Windows OS Vulnerabilities now include a `resolved_in_version` in the `/os_versions` API response.
  - Fixed an issue where software from a Parallels VM would incorrectly appear as the host's software.
  - Implemented permission checks for software and software titles.
  - Fixed software title aggregation when triggering vulnerability scans.

### Bug fixes and improvements
  - Updated text and style across the app for consistency and clarity.
  - Improved UI for the view disk encryption key, host details activity card, and "Add hosts" modal.
  - Addressed a bug where updating the search field caused unwanted loss of focus.
  - Corrected alignment bugs on empty table states for software details.
  - Updated URL query parameters to reset when switching tabs.
  - Fixed device page showing invalid date for the last restarted.
  - Fixed visual display issues with chevron right icons on Chrome.
  - Fixed Windows vulnerabilities without exploit/severity from crashing the software page.
  - Fixed issues with checkboxes in hidden modals and long enroll secrets overlapping action buttons.
  - Fixed a bug with built-in platform labels.
  - Fixed enroll secret error messaging showing secret in cleartext.
  - Fixed various UI bugs including disk encryption key input icons, alignment issues, and dropdown menus.
  - Fixed dropdown behavior in administrative settings and software title/version tables.
  - Fixed various UI and style bugs, including issues with long OS names causing table render issues.
  - Fixed a bug where checkboxes within a hidden modal were not correctly hidden.
  - Fixed vulnerable software dropdown from switching back to all teams.
  - Fixed wall_time to report in milliseconds for consistency with other query performance stats.
  - Fixed generating duplicate activities when locking or unlocking a host with scripts disabled.
  - Fixed how errors are reported to APM to avoid duplicates and improve stack trace accuracy.

## Fleet 4.44.1 (Feb 13, 2024)

### Bug fixes

* Fixed a bug where long enrollment secrets would overlap with the action buttons on top of them.
* Fixed a bug that caused OS Settings to never be verified if the MySQL config of Fleet's database had 'only_full_group_by' mode enabled (enabled by default).
* Ensured policy names are now unique per team, allowing different teams to have policies with the same name.
* Fixed the visual display of chevron right icons on Chrome.
* Renamed the 'mdm_windows_configuration_profiles' and 'mdm_apple_configuration_profiles' 'updated_at' field to 'uploaded_at' and removed the automatic setting of the value, setting it explicitly instead.
* Fixed a small alignment bug in the setup flow.
* Improved the validation of Windows profiles to prevent errors when delivering the profiles to the hosts. If you need to embed a nested XML structure (for example, for Wi-Fi profiles), you can either:
 - Escape the XML.
 - Use a wrapping `<![CDATA[ ... ]]>` element.
* Fixed an issue where an inaccurate message was returned after running an asynchronous (queued) script.
* Fixed URL query parameters to reset when switching tabs.
* Fixed the vulnerable software dropdown from switching back to all teams.
* Added fleetctl gitops command:
 - Synchronize Fleet configuration with the provided file. This command is intended to be used in a GitOps workflow.
* Updated the response for 'GET /api/v1/fleet/hosts/:id/activities/upcoming' to include the count of all upcoming activities for the host.
* Fixed an issue where software from a Parallels VM on a MacOS host would show up in Fleet as if it were the host's software.
* Removed unnecessary nested database transactions in batch-setting of MDM profiles.
* Added count of upcoming activities to host vitals UI.

## Fleet 4.44.0 (Jan 31, 2024)

### Changes

* **Endpoint operations**:
  - Removed rate-limiting from `/api/fleet/orbit/ping` and `/api/fleet/device/ping` endpoints.
  - For Windows hosts, fleetd now uses Windows Credential Manager for enroll secret.
  - For macOS hosts, fleetd stores and retrieves enroll secret from macOS keychain for non-MDM flow.
  - Query reports feature now supports a custom `pack_delimiter` in agent settings.
  - Packaged `fleetctl` for macOS as a universal binary (native support for both amd64 and arm64 architectures).
  - Added new flow for `fleetctl package --type=msi` on macOS using arm64 processor.
  - Teams can now configure their own host expiry settings.
  - Added UI for host details activity card.
  - Added `host_count_updated_at` to policy API responses.
  - Added "Run script" action to host details page.
  - Created the "script ran" activity linked to its host.
  - Updated host details page and `GET /api/v1/fleet/hosts/:id` endpoint so that failing policies are listed first.

* **Device management (MDM)**:
  - Added new endpoints `GET /api/v1/fleet/mdm/manual_enrollment_profile` and scripts related endpoints (`/hosts/:id/activity`, `/hosts/:id/activity/upcoming`).
  - Added support for label-based MDM profiles reconciliation.
  - Improved MDM migration puppet module.
  - Added Windows scripts for MDM unenrollment and fleetd removal.
  - Added the profile's `labels` object to MDM profiles response payload.
  - Updated UI with ability to target MDM profiles by label.
  - Added ability to configure custom `configuration_web_url` values in DEP profile.
  - Fixed a bug causing MDM SSO to fail with certain configurations.
  - Fixed queries reporting inconsistent MDM enrollment status in Windows.

* **Vulnerability management**:
  - Added support for detecting operating system vulnerabilities for macOS and Windows.
  - Corrected Windows OS false negative for multiple OS build remediations.
  - Fixed issue with incorrect `resolved_in_version` for vulnerabilities.

### Bug fixes and improvements

  - Added "No report" text for query results not saved in Fleet.
  - Updated forms across the UI for consistent styling.
  - Improved UX for globally enabling/disabling SSO.
  - Added new consistent header styling across the app.
  - Clearer browser page titles and CTAs for Observer+.
  - Updated logging destination failure response to return a 4xx error instead of 500.
  - Addressed issues with query reports and host expiry settings.
  - Resolved platform compatibility checker issues with deprecated osquery tables.
  - Updated Go to version 1.21.6.
  - osquery flag validation updated for osquery 5.11.
  - Fixed validation and error handling for `/api/fleet/orbit/device_token` and other endpoints.
  - Fixed UI bugs in script functionality, side navigation content headers, and premium message alignment.
  - Fixed a bug in searching for hosts by email addresses.
  - Fixed issues with sticky errors in fleetd-chrome after querying privacy_preferences table.
  - Fixed a bug where Munki issues section was incorrectly displayed.
  - Fixed OS compatibility calculation for certain queries.
  - Fixed a bug where capital characters would not match labels containing them.
  - Fixed bug in manage hosts UI where changing the dropdown filter did not clear OS settings filter.
  - Fixed a bug in `fleetctl` where `--context` and `--debug` flags were not allowed after certain commands.
  - Fixed a bug where the UUID for Windows updates profiles was missing the `"w"` prefix.
  - Fixed a UI bug on the controls page in team targeting forms.
  - Fixed a bug where policy automations when saved were resetting automations on other pages.

## Fleet 4.43.3 (Jan 23, 2024)

### Bug fixes

* Fixed incorrect padding on the my device page.

## Fleet 4.43.2 (Jan 22, 2024)

### Bug fixes

* Improved HTTP client used by `fleetctl` and `fleetd` to prevent errors for 204 responses.
* Added free tier UI state to OS updates and setup experience pages.
* Added warning/info messages when downgrading/upgrading `fleetd` or OSQuery.
* Updated links to an expired osquery Slack invitation to go to the support page on the Fleet website.
* Cleaned settings styling.
* Created consistent loading states when using search filter.
* Fixed center styling for empty states. For `software/titles` and `software/versions` endpoints, the
  `browser` property is no longer included in the response when empty.
* Fixed the Windows MDM polling interval so that enrolled devices check-in regularly with Fleet to look for pending MDM-related actions.
* Fixed missing empty members SVG by fixing SVG IDs.
* Fixed a bug that caused the software/titles page to error.
* Fixed 2 vulnerability false positives on Microsoft Teams on MacOS.
* Fixed bug in CIS policy: Ensure an Inactivity Interval of 20 Minutes Or Less for the Screen Saver Is Enabled.

## Fleet 4.43.1 (Jan 15, 2024)

### Bug fixes

* Fixed bug where script results would sometimes show the wrong error message when a user attempts
  to run a script on a host that has scripts disabled.
* Fixed an issue with SCEP endpoints sending back 500 status codes. Should return 400 now if bad
  data is sent to SCEP API.
* Fixed text and icon alignment UI bug.
* Fixed message for script execution timeout.
* Fixed failed scripts showing the wrong error.

## Fleet 4.43.0 (Jan 9, 2024)

### Changes

* **Endpoint operations**:
  - Added new `POST /api/v1/fleet/queries/:id/run` endpoint for synchronous live queries.
  - Added `PUT /api/fleet/orbit/device_mapping` and `PUT /api/v1/fleet/hosts/{id}/device_mapping` endpoints for setting or replacing custom email addresses.
  - Added experimental `--end-user-email` flag to `fleetctl package` for `.msi` installer bundling.
  - Added `host_count_updated_at` to policy API responses.
  - Added ability to query by host display name via list hosts endpoint.
  - Added `gigs_total_disk_space` to host endpoint responses.
  - Added ability to remotely configure `fleetd` update channels in agent options (Fleet Premium only, requires `fleetd` >= 1.20.0).
  - Improved error message for osquery log write failures.
  - Protect live query performance by limiting results per live query.
  - Improved error handling and validation for `/api/fleet/orbit/device_token` and other endpoints.

* **Device management (MDM)**:
  - Added check for custom end user email fields in enrollment profiles.
  - Modified hosts and labels endpoints to include only user-defined Windows MDM profiles.
  - Improved profile verification logic for 'pending' profiles.
  - Updated enrollment process so that `fleetd` auto-installs on Apple hosts enabling MDM features manually.
  - Extended script execution timeout to 5 minutes.
  - Extended Script disabling functionality to various script endpoints and `fleetctl`.

### Bug fixes and improvements
  - Fix profiles incorrectly being marked as "Failed". 
    - **NOTE**: If you are using MDM features and have already upgraded to v4.42.0, you will need to take manual steps to resolve this issue. Please [follow these instructions](https://github.com/fleetdm/fleet/issues/15725) to reset your profiles. 
  - Added tooltip to policies page stating when policy counts were last updated.
  - Added bold styling to profile name in custom profile activity logs.
  - Implemented style tweaks to the nudge preview on OS updates page.
  - Updated sort query results and reports case sensitivity and default to sorting.
  - Added disk size indication when disk is full. 
  - Replaced 500 error with 409 for token conflicts with another host.
  - Fixed script output text formatting.
  - Fixed styling issues in policy automations modal and nudge preview on OS updates page.
  - Fixed loading spinner not appearing when running a script on a host.
  - Fixed duplicate view all hosts link in disk encryption table.
  - Fixed tooltip text alignment UI bug.
  - Fixed missing 'Last restarted' values when filtering hosts by label.
  - Fixed broken link on callout box on host details page. 
  - Fixed bugs in searching hosts by email addresses and filtering by labels.
  - Fixed a bug where the host details > software > munki issues section was sometimes displayed erroneously.
  - Fixed a bug where OS compatibility was not correctly calculated for certain queries.
  - Fixed issue where software title aggregation was not running during vulnerability scans.
  - Fixed an error message bug for password length on new user creation.
  - Fixed a bug causing misreporting of vulnerability scanning status in analytics.
  - Fixed issue with query results reporting after discard data is enabled.
  - Fixed a bug preventing label selection while the label search field was active.
  - Fixed bug where `fleetctl` did not allow placement of `--context` and `--debug` flags following certain commands.
  - Fixed a validation bug allowing `overrides.platform` to be set to `null`.
  - Fixed `fleetctl` issue with creating a new query when running a query by name.
  - Fixed a bug that caused vulnerability scanning status to be misreported in analytics.
  - Fixed CVE tooltip bullets on the software page.
  - Fixed a bug that didn't allow enabling team disk encryption if macOS MDM was not configured.

## Fleet 4.42.0 (Dec 21, 2023)

### Changes

* **Endpoint operations**:
  - Added `fleet/device/{token}/ping` endpoint for agent token checks.
  - Added `GET /hosts/{id}/health` endpoint for host health data.
  - Added `--host-identifier` option to fleetd for enrolling with a random identifier.
  - Added capability to look up hosts based on IdP email.
  - Updated manage hosts UI to filter hosts by `software_version_id` and `software_title_id`.
  - Added ability to filter hosts by `software_version_id` and `software_title_id` in various endpoints.
  - **NOTE:**: Database migrations may take up to five minutes to complete based on number of software items. 
  - Live queries now collect and display updated stats.
  - Live query stats are cleared when query SQL is modified.
  - Added UI features to incorporate new live query stats.
  - Improved host query reports and host detail query tab UI.
  - Added firehose delivery addon update for improved data handling.

* **Vulnerability management**:
  - Added `GET software/versions` and `GET software/versions/{id}` endpoints for software version management.
  - Deprecated `GET software` and `GET software/{id}` endpoints.
  - Added new software pages in Fleet UI, including software titles and versions.
  - Resolved scan error during OVAL vulnerability processing.

* **Device management (MDM)**:
  - Removed the `FLEET_DEV_MDM_ENABLED` feature flag for Windows MDM.
  - Enabled `fleetctl` to configure Windows MDM profiles for teams and "no team".
  - Added database tables to support the Windows profiles feature.
  - Added support to configure Windows OS updates requirements.
  - Introduced new MDM profile endpoints: `POST /mdm/profiles`, `DELETE /mdm/profiles/{id}`, `GET /mdm/profiles/{id}`, `GET /mdm/profiles`, `GET /mdm/profiles/summary`.
  - Added validation to disallow custom MDM profiles with certain names.
  - Added deployment of Windows OS updates settings to targeted hosts.
  - Changed the Apple profiles ID to a prefixed UUID format.
  - Enabled targeting hosts by serial number in `fleetctl run-script` and `fleetctl mdm run-command`.
  - Added UI for uploading, deleting, downloading, and viewing Windows custom MDM profiles.

### Bug fixes and improvements

  - Updated Go version to 1.21.5.
  - Query reports now only show results for hosts with user permissions.
  - Global observers can now see all queries regardless of the observerCanRun value.
  - Added whitespace rendering in policy descriptions and resolutions.
  - Added truncation to dropdown options in query tables documentation.
  - `POST /api/v1/fleet/scripts/run/sync` timeout now returns error code 408 instead of 504.
  - Fixed possible deadlocks in `software` data ingestion and `host_batteries` upsert.
  - Fixed button text wrapping in UI for Settings > Integrations > MDM.
  - Fixed a bug where opening a modal on the Users page reset the table to the first page.
  - Fixed a bug preventing label selection while the label search field was active.
  - Fixed issues with UI loading indicators and placeholder texts.
  - Fixed a fleetctl issue where running a query by name created a new query instead of using the existing one.
  - Fixed `installed_from_dep` in `mdm_enrolled` activity for DEP device re-enrollment.
  - Fixed a bug in line breaks affecting UI functionality.
  - Fixed Syncml cmd data support for raw data.
  - Added "copied!" message to the copy button on inputs.
  - Fixed an edge case where caching could lead to lost organization settings in multiple instance scenarios.
  - Fixed `GET /hosts/{id}/health` endpoint reporting.
  - Fixed validation bugs allowing `overrides.platform` field to be set to `null`.
  - Fixed an issue with policy counts showing 0 post-upgrade.

## Fleet 4.41.1 (Dec 7, 2023)

### Bug fix

* Fixed logging of results for scheduled queries configured outside of Fleet when `server_settings.query_reports_disabled` is set to `true`.

## Fleet 4.41.0 (Nov 28, 2023)

### Changes

* **Endpoint operations**:
  - Enhanced `fleetctl` and API to support PowerShell (.ps1) scripts.
  - Updated several API endpoints to support `os_settings` filter, including Windows profiles status.
  - Enabled `after` parameter for improved pagination in various endpoints.
  - Improved the `fleet/queries/run` endpoint with better error handling.
  - Increased frequency of metrics reporting from Fleet servers to daily.
  - Added caching for policy results in MySQL for faster operations.

* **Device management (MDM)**:
  - Added database tables for Windows profiles support.
  - Added validation for WSTEP certificate and key pair before enabling Windows MDM.

* **Vulnerability management**:
  - Fleet now uses NVD API 2.0 for CVE information download.
  - Added support for JetBrains application vulnerability data.
  - Tightened software matching to reduce false positives.
  - Stopped reporting Atom editor packages in software inventory.
  - Introduced support for Windows PowerShell scripts in the UI.
  
* **UI improvements**:
  - Updated activity feed for better communication around JIT-provisioned user logins.
  - Query report now displays the host's display name instead of the hostname.
  - Improved UI components like the manage page's label filter and edit columns modal.
  - Enabled all sort headers in the UI to be fully clickable.
  - Removed the creation of OS policies from a host's operating system in the UI.
  - Ensured correct settings visibility in the Settings > Advanced section.

### Bug fixes

  - Fixed long result cell truncation in live query results and query reports.
  - Fixed a Redis cluster mode detection issue for RedisLabs hosted instances.
  - Fixed a false positive vulnerability report for Citrix Workspace.
  - Fixed an edge case sorting bug related to the `last_restarted` value for hosts.
  - Fixed an issue with creating .deb installers with different enrollment keys.
  - Fixed SMTP configuration validation issues for TLS-only servers.
  - Fixed caching of team MDM configurations to improve performance at scale.
  - Fixed delete pending issue during orbit.exe installation.
  - Fixed a bug causing the disk encryption key banner to not display correctly.
  - Fixed various error code inconsistencies across endpoints.
  - Fixed filtering hosts with invalid team_id now returns a 400 error.
  - Fixed false positives in software matching for similar names.

## Fleet 4.40.0 (Nov 3, 2023)

### Changes

* **Endpoint operations**:
  - New tables added to the fleetd extension: app_icons, falconctl_options, falcon_kernel_check, cryptoinfo, cryptsetup_status, filevault_status, firefox_preferences, firmwarepasswd, ioreg, and windows_updates.
  - CIS support for Windows 10 is updated to the lates CIS document CIS_Microsoft_Windows_10_Enterprise_Benchmark_v2.0.0.

* **Device management (MDM)**:
  - Introduced support for MS-MDM management protocol.
  - Added a host detail query for Windows hosts to ingest MDM device id and updated the Windows MDM device enrollment flow.
  - Implemented `--context` and `--debug` flags for `fleetctl mdm run-command`.
  - Support added for `fleetctl mdm run-command` on Windows hosts.
  - macOS hosts with MDM features via SSO can now run `sudo profiles renew --type enrollment`.
  - Introduced `GET mdm/commandresults` endpoint to retrieve MDM command results for Windows and macOS.
  - `fleetctl get mdm-command-results` now uses the new above endpoint.
  - Added `POST /fleet/mdm/commands/run` platform-agnostic endpoint for MDM commands.
  - Introduced API for recent Windows MDM commands via `fleetctl` and the API.

* **Vulnerability management**:
  - Added vulnerability data support for JetBrains apps with similar names (e.g., IntelliJ IDEA.app vs. IntelliJ IDEA Ultimate.app).
  - Apple Rapid Security Response version added to macOS host details (requires osquery v5.9.1 on macOS devices).
  - For ChromeOS hosts, software now includes chrome extensions.
  - Updated vulnerability processing to omit software without versions.
  - Resolved false positives in vulnerabilities for Chrome and Firefox extensions.

* **UI improvements**:
  - Fleet tables in UI reset rows upon filter/search/page changes.
  - Improved handling when deleting a large number of hosts; operations now continue in the background after 30 seconds.
  - Added the ability for Observers and Observer+ to view policy resolutions.
  - Improved app settings clarity for premium users regarding usage statistics.
  - UI buttons for live queries or policies are now disabled with a tooltip if live queries are globally turned off.
  - Observers and observer+ can now run existing policies in the UI.

### Bug fixes and improvements

* **REST API**:
  - Overhauled REST API input validation for several endpoints (hosts, carves, users).
  - Validation error status codes switched from 500 to 400 for clarity.
  - Numerous new validations added for policy details, os_name/version, etc.
  - Addressed issues in /fleet/sso and /mdm/apple/enqueue endpoints.
  - Updated response codes for several other endpoints for clearer error handling.

* **Logging and debugging**:
  - Updated Apple Business Manager terms logging behavior.
  - Refined the copy of the ABM terms banner for better clarity.
  - Addressed a false positive CVE detection on the `certifi` python package.
  - Fixed a logging issue with Fleet's Cloudflare WARP software version ingestion for Windows.

* **UI fixes**:
  - Addressed UI bugs for the "Turn off MDM" action display and issues with the host details page's banners.
  - Fixed narrow viewport EULA display issue on the Windows TOS page.
  - Rectified team dropdown value issues and ensured consistent help text across query and policy creation forms.
  - Fixed issues when applying config changes without MDM features enabled.

* **Others**:
  - Removed the capability for Premium customers to disable usage statistics. Further information provided in the Fleet documentation.
  - Retired creating OS policies from host OSes in the UI.
  - Addressed issues in Live Queries with the POST /fleet/queries/run endpoint.
  - Introduced database migrations for Windows MDM command tables.

## Fleet 4.39.0 (Oct 19, 2023)

### Changes

* Added ability to store results of scheduled queries:
  - Will store up to 1000 results for each scheduled query. 
  - If the number of results for a scheduled query is below 1000, then the results will continuously get updated every time the hosts send results to Fleet.
  - Introduced `server_settings.query_reports_disabled` field in global configuration to disable this feature.
  - New API endpoint: `GET /api/_version_/fleet/queries/{id}/report`.
  - New field `discard_data` added to API queries endpoints for toggling report storage for a query. For yaml configurations, use `discard_data: true` to disable result storage.
  - Enhanced osquery result log validation.
  - **NOTE:** This feature enables storing more query data in Fleet. This may impact database performance, depending on the number of queries, their frequency, and the number of hosts in your Fleet instance. For large deployments, we recommend monitoring your database load while gradually adding new query reports to ensure your database is sized appropriately.

* Added scripts tab and table for host details page.

* Added support to return the decrypted disk encryption key of a Windows host.

* Added `GET /hosts/{id}/scripts` endpoint to retrieve status details of saved scripts for a host.

* Added `mdm.os_settings` to `GET /api/v1/hosts/{id}` response.

* Added `POST /api/fleet/orbit/disk_encryption_key` endpoint for Windows hosts to report bitlocker encryption key.

* Added activity logging for script operations (add, delete, edit).

* Added UI for scripts on the controls page.

* Added API endpoints for script management and updated existing ones to accommodate saved script ID.

* Added `GET /mdm/disk_encryption/summary` endpoint for disk encryption summaries for macOS and Windows.

* Added `os_settings` and `os_settings_disk_encryption` filters to various `GET` endpoints for host filtering based on OS settings.

* Enhanced `GET hosts/:id` API response to include more detailed disk encryption data for device client errors.

* Updated controls > disk encryption and host details page to include Windows bitlocker information.

* Improved styling for host details/device user failing policies display.

* Disabled multicursor editing for SQL editors.

* Deprecated `mdm.macos_settings.enable_disk_encryption` in favor of `mdm.enable_disk_encryption`.

* Updated Go version to 1.21.3.

### Bug fixes

* Fixed script content and output formatting issues on the scripts detail modal.

* Fixed a high database load issue in the Puppet match endpoint.

* Fixed setup flows background not covering the entire viewport when resized to some sizes.

* Fixed a bug affecting OS settings information retrieval regarding disk encryption status for Windows hosts.

* Fixed SQL parameters used in the `/api/latest/fleet/labels/{labelID}/hosts` endpoint for certain query parameters, addressing issue 13809.

* Fixed Python's CVE-2021-42919 false positive on macOS which should only affect Linux.

* Fixed a bug causing DEP profiles to sometimes not get assigned correctly to hosts.

* Fixed an issue in the bulk-set of MDM Apple profiles leading to excessive placeholders in SQL.

* Fixed max-height display issue for script content and output in the script details modal.

## Fleet 4.38.1 (Oct 5, 2023)

### Bug Fixes

* Fixed a bug that would cause live queries to stall if a detail query override was set for a team.

## Fleet 4.38.0 (Sep 25, 2023)

### Changes

* Updated MDM profile verification so that an install profile command will be retried once if the command resulted in an error or if osquery cannot confirm that the expected profile is installed.

* Ensured post-enrollment commands are sent to devices assigned to Fleet in ABM.

* Ensured hosts assigned to Fleet in ABM come back to pending to the right team after they're deleted.

* Added `labels` to the fleetd extensions feature to allow deploying extensions to hosts that belong to certain labels.

* Changed fleetd Windows extensions file extension from `.ext` to `.ext.exe` to allow their execution on Windows devices (executables on Windows must end with `.exe`).

* Surfaced chrome live query errors to Fleet UI (including errors for specific columns while maintaining successful data in results).

* Fixed delivery of fleetd extensions to devices to only send extensions for the host's platform.

* (Premium only) Added `resolved_in_version` to `/fleet/software` APIs pulled from NVD feed.

* Added database migrations to create the new `scripts` table to store saved scripts.

* Allowed specifying `disable_failing_policies` on the `/api/v1/fleet/hosts/report` API endpoint for increased performance. This is useful if the user is not interested in counting failed policies (`issues` column).

* Added the option to use locally-installed WiX v3 binaries when generating the Fleetd installer for Windows on a Windows machine.

* Added CVE descriptions to the `/fleet/software` API.

* Restored the ability to click on and select/copy text from software bundle tooltips while maintaining the abilities to click the software's name to get more details and to click anywhere else in the row to view all hosts with that software installed.

* Stopped 1password from overly autofilling forms.

* Upgraded Go version to 1.21.1.

### Bug Fixes

* Fixed vulnerability mismatch between the flock browser and the discoteq/flock binary.

* Fixed v4.37.0 performance regressions in the following API endpoints:
  * `/api/v1/fleet/hosts/report`
  * `/api/v1/fleet/hosts` when using `per_page=0` or a large number for `per_page` (in the thousands).

* Fixed script content and output formatting on the scripts detail modal.

* Fixed wrong version numbers for Microsoft Teams in macOS (from invalid format of the form `1.00.XYYYYY` to correct format `1.X.00.YYYYY`).

* Fixed false positive CVE-2020-10146 found on Microsoft Teams.

* Fixed CVE-2013-0340 reporting as a valid vulnerability due to NVD recommendations.

* Fixed save button for a new policy after newly creating another policy.

* Fixed empty query/policy placeholders.

* Fixed used by data when filtering hosts by labels.

* Fixed small copy and alignment issue with status indicators in the Queries page Automations column.

* Fixed strict checks on Windows MDM Automatic Enrollment.

* Fixed software vulnerabilities time ago column for old CVEs.

## Fleet 4.37.0 (Sep 8, 2023)

### Changes

* Added `/scripts/run` and `scripts/run/sync` API endpoints to send a script to be executed on a host and optionally wait for its results.

* Added `POST /api/fleet/orbit/scripts/request` and `POST /api/fleet/orbit/scripts/result` Orbit-specific API endpoints to get a pending script to execute and send the results back, and added an Orbit notification to let the host know it has scripts pending execution.

* Improved performance at scale when applying hundreds of policies to thousands of hosts via `fleetctl apply`.
  - IMPORTANT: In previous versions of Fleet, there was a performance issue (thundering herd) when applying hundreds of policies on a large number of hosts. To avoid this, make sure to deploy this version of Fleet, and make sure Fleet is running for at least 1h (or the configured `FLEET_OSQUERY_POLICY_UPDATE_INTERVAL`) before applying the policies.

* Added pagination to the policies API to increase response time.

* Added policy count endpoints to support pagination on the frontend.

* Added an endpoint to report `fleetd` errors.

* Added logic to report errors during MDM migration.

* Added support in fleetd to execute scripts and send back results (disabled by default).

* Added an activity log when script execution was successfully requested.

* Automatically set the DEP profile to be the same as "no team" (if set) for teams created using the `/match` endpoint (used by Puppet).

* Added JumpCloud to the list of well-known MDM solutions.

* Added `fleetctl run-script` command.

* Made all table links right-clickable.

* Improved the layout of the MDM SSO pages.

* Stored user email when a user turned on MDM features with SSO enabled.

* Updated the copy and image displayed on the MDM migration modal.

* Upgraded Go to v1.19.12.

* Updated the macadmins/osquery-extension to v0.0.15.

* Updated nanomdm dependency.

### Bug Fixes

* Fixed a bug where live query UI and export data tables showed all returned columns.

* Fixed a bug where Jira and/or Zendesk integrations were being removed when an unrelated setting was changed.

* Fixed software ingestion to not re-insert software when incoming fields from hosts were longer than what Fleet supports. This bug caused some CVEs to be reported every time the vulnerability cron ran.
  - IMPORTANT: After deploying this fix, the vulnerability cron will report the CVEs one last time, and subsequent cron runs will not report the CVE (as expected).

* Fixed duplicate policy names in `ee/cis/win-10/cis-policy-queries.yml`.

* Fixed typos in policy queries in the Windows CIS policies YAML (`ee/cis/win-10/cis-policy-queries.yml`).

* Fixed a bug where query stats (aka `Performance impact`) were not being populated in Fleet.

* Added validation to `fleetctl apply` for duplicate policy names in the YAML file and attempting to change the team of an existing policy.

* Optimized host queries when using policy statuses.

* Changed the authentication method during Windows MDM enrollment to use `LoadHostByOrbitNodeKey` instead of `HostByIdentifier`.

* Fixed alignment on long label names on host details label filter dropdown.

* Added UI for script run activity and script details modal.

* Fixed queries navigation bar bug where if in query detail, you could not navigate back to the manage queries table.

* Made policy resolutions that include URLs clickable in the UI.

* Fixed Fleet UI custom query frequency display.

* Fixed live query filter icon and various other live query icons.

* Fixed Fleet UI tabs highlight while tabbing but not on multiple clicks.

* Fixed double scrollbar bug on dashboard page.

## Fleet 4.36.0 (Aug 17, 2023)

* Added the `fleetctl upgrade-packs` command to migrate 2017 packs to the new combined schedule and query concept.

* Updated `fleetctl convert` to convert packs to the new combined schedule and query format.

* Updated the `POST /mdm/apple/profiles/match` endpoint to set the bootstrap package and enable end user authentication settings for each new team created via the endpoint to the corresponding values specified in the app config as of the time the applicable team is created.

* Added enroll secret for a new team created with `fleetctl apply` if none is provided.

* Improved SQL autocomplete with dynamic column, table names, and shown metadata.

* Cleaned up styling around table search bars.

* Updated MDM profile verification to fix issue where profiles were marked as failed when a host 
is transferred to a newly created team that has an identical profile as an older team.

* Added windows MDM automatic enrollment setup pages to Fleet UI.

* (Beta) Allowed configuring Windows MDM certificates using their contents.

* Updated the icons on the dashboard to new grey designs.

* Ensured DEP profiles are assigned even for devices that already exist and have an op type = "modified".

* Disabled save button for invalid query or policy SQL & missing name.

* Users with no global or team role cannot access the UI.

* Text cells truncate with ellipses if longer than column width.
  
**Bug Fixes:**

* Fixed styling issue of the active settings tab.

* Fixed response status code to 403 when a user cannot change their password either because they were not requested to by the admin or they have Single-Sign-On (SSO) enabled.

* Fixed issues with end user migration flow.

* Fixed login form cut off when viewport is too short.

* Fixed bug where `os_version` endpoint returned 404 for `no teams` on controls page.

* Fixed delays applying profiles when the Puppet module is used in distributed scenarios.

* Fixed a style issue in the filter host by status dropdown.

* Fixed an issue when a user with `gitops` role was used to validate a configuration with `fleetctl apply --dry-run`.

* Fixed jumping text on the host page label filter dropdown at low viewport widths.

## Fleet 4.35.2 (Aug 10, 2023)

* Fixed a bug that set a wrong Fleet URL in Windows installers.

## Fleet 4.35.1 (Aug 4, 2023)

* Fixed a migration to account for columns with NULL values as a result of either creating schedules via the API without providing all values or by a race condition with database replicas.

* Fixed a bug that occurred when a user tried to create a custom query from the "query" action on a host's details page.

## Fleet 4.35.0 (Jul 31, 2023)

* Combined the query and schedule features to provide a single interface for creating, scheduling, and tweaking queries at the global and team level.

* Merged all functionality of the schedule page into the queries page.

* Updated the save query modal to include scheduling-related fields.

* Updated queries table schema to allow storing scheduling information and configuration in the queries table.

* Users now able to manage scheduled queries using automations modal.

* The `osquery/config` endpoint now includes scheduled queries for the host's team stored in the `queries` table.

* Query editor now includes frequency and other advanced options.

* Updated macOS MDM setup UI in Fleet UI.

* Changed how team assignment works for the Puppet module, for more details see the [README](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/README.md).

* Allow the Puppet module to read different Fleet URL/token combinations for different environments.

* Updated server logging for webhook requests to mask URL query values if the query param name includes "secret", "token", "key", "password".

* Added support for Azure JWT tokens.

* Set `DeferForceAtUserLoginMaxBypassAttempts` to `1` in the default FileVault profile installed by Fleet.

* Added dark and light mode logo uploads and show the appropriate logo to the macOS MDM migration flow.

* Added MSI installer deployement support through MS-MDM.

* Added support for Windows MDM STS Auth Endpoint.

* Added support for installing Fleetd after enrolling through Azure account.

* Added support for MDM TOS endpoint.

* Updated the "Platforms" column to the more explicit "Compatible with".

* Improved delivery of Apple MDM profiles by not re-sending `InstallProfile` commands if a host switches teams but the profile contents are the same.

* Improved error handling and messaging of SSO login during AEP(DEP) enrollments.

* Improved the reporting of the Puppet module to only report as changed profiles that actually changed during a run.

* Updated ingestion of host detail queries for MDM so hosts that report empty results are counted as "Off".

* Upgraded Go version to v1.19.11.

* If a policy was defined with an invalid query, the desktop endpoint now counts that policy as a failed policy.

* Fixed issue where Orbit repeatedly tries to launch Nudge in the event of a launch error.

* Fixed Observer + should be able to run any query by clicking create new query.

* Fixed the styling of the initial setup flow.

* Fixed URL used to check Gravatar network availability.

## Fleet 4.34.1 (Jul 14, 2023)

* Fixed Observer+ not being able to run some queries.

* If a policy was defined with an invalid query, the desktop endpoint should count that policy as a failed policy.

## Fleet 4.34.0 (Jul 11, 2023)

* Added execution of programmatic Windows MDM enrollment on eligible devices when Windows MDM is enabled.

* Microsoft MDM Enrollment Protocol: Added support for the RequestSecurityToken messages.

* Microsoft MDM Enrollment Protocol: Added support for the DiscoveryRequest messages.

* Microsoft MDM Enrollment Protocol: Added support for the GetPolicies messages.

* Added `enabled_windows_mdm` and `disabled_windows_mdm` activities when a user turns on/off Windows MDM.

* Added support to enable and configure Windows MDM and to notify devices that are able to programmatically enroll.

* Added ability to turn Windows MDM on and off from the Fleet UI.

* Added enable and disable Windows MDM activity UI.

* Updated MDM detail query ingestion to switch MDM profiles from "verifying" or "verified" status to "failed" status when osquery reports that this profile is not installed on the host.

* Added notification and execution of programmatic Windows MDM unenrollment on eligible devices when Windows MDM is disabled.

* Added the `FLEET_DEV_MDM_ENABLED` environment variable to enable the Windows MDM feature during its development and beta period.

* Added the `mdm_enabled` feature flag information to the response payload of the `PATCH /config` endpoint.

* When creating a PolicySpec, return the proper HTTP status code if the team is not found.

* Added CPEMatchingRule type, used for correcting false positives caused by incorrect entries in the NVD dataset.

* Optimized macOS CIS query "Ensure Appropriate Permissions Are Enabled for System Wide Applications" (5.1.5).

* Updated macOS CIS policies 5.1.6 and 5.1.7 to use a new fleetd table `find_cmd` instead of relying on the osquery `file` table to improve performance.

* Implemented the privacy_preferences table for the Fleetd Chrome extension.

* Warnings in fleetctl now go to stderr instead of stdout.

* Updated UI for transferred hosts activity items.

* Added Organization support URL input on the setting page organization info form.

* Added improved ABM 400 error message to the UI.

* Hide any osquery tables or columns from Fleet UI that has hidden set to true to match Fleet website.

* Ignore casing in SAML response for display name. For example the display name attribute can be provided now as `displayname` or `displayName`.

* Provide feedback to users when `fleetctl login` is using EMAIL and PASSWORD environment variables.

* Added a new activity `transferred_hosts` created when hosts are transferred to a new team (or no team).

* Added milliseconds to the timestamp of auto-generated team name when creating a new team in `GET /mdm/apple/profiles/match`.

* Improved dashboard loading states.

* Improved UI for selecting targets.

* Made sure that all configuration profiles and commands are sent to devices if MDM is turned on, even if the device never turned off MDM.

* Fixed bug when reading filevault key in osquery and created new Fleet osquery extension table to read the file directly rather than via filelines table.

* Fixed UI bug on host details and device user pages that caused the software search to not work properly when searching by CVE.

* Fixed not validating the schema used in the Metadata URL.

* Fixed improper HTTP status code if SMTP is invalid.

* Fixed false positives for iCloud on macOS.

* Fixed styling of copy message when copying fields.

* Fixed a bug where an empty file uploaded to `POST /api/latest/fleet/mdm/apple/setup/eula` resulted in a 500; now returns a 400 Bad Request.

* Fixed vulnerability dropdown that was hiding if no vulnerabilities.

* Fixed scroll behavior with disk encryption status.

* Fixed empty software image in sandbox mode.

* Fixed improper HTTP status code when `fleet/forgot_password` endpoint is rate limited. 

* Fixed MaxBurst limit parameter for `fleet/forgot_password` endpoint.

* Fixed a bug where reading from the replica would not read recent writes when matching a set of MDM profiles to a team (the `GET /mdm/apple/profiles/match` endpoint).

* Fixed an issue that displayed Nudge to macOS hosts if MDM was configured but MDM features weren't turned on for the host.

* Fixed tooltip word wrapping on the error cell in the macOS settings table.

* Fixed extraneous loading spinner rendering on the software page.

* Fixed styling bug on setup caused by new font being much wider.

## Fleet 4.33.1 (Jun 20, 2023)

* Fixed ChromeOS add host instructions to use variable Fleet URL.

## Fleet 4.33.0 (Jun 12, 2023)

* Upgraded Go version to 1.19.10.

* Added support for ChromeOS devices.

* Added instructions to inform users how to add ChromeOS hosts.

* Added ChromeOS details to the dashboard, manage hosts, and host details pages.

* Added ability for users to create policies that target ChromeOS.

* Added built-in label for ChromeOS.

* Added query to fill in `device_mapping` from ChromeOS hosts.

* Improved the performance of live query results rendering to address usability issues when querying tens of thousands of hosts.

* Reduced size of live query websocket message by removing unused host data.

* Added the `POST /fleet/mdm/apple/profiles/preassign` endpoint to store profiles to be assigned to a host for subsequent matching with an existing (or new) team.

* Added the `POST /fleet/mdm/apple/profiles/match` endpoint to match pre-assigned profiles to an existing team or create one if needed, and assign the host to that team.

* Updated `GET /mdm/apple/profiles` endpoint to return empty array instead of null if no profiles are found.

* Improved ingestion of MDM devices from ABM:
  - If a device's operation_type is `modified`, but the device doesn't exist in Fleet yet, a DEP profile will be assigned to the device and a new record will be created in Fleet.
  - If a device's operation_type is `deleted`, the device won't be prompted to migrate to Fleet if the feature has been configured.

* Added "Verified" profile status for profiles verified with osquery.

* Added "Action required" status for disk encryption profile in UI for host details and device user pages.

* Added UI for the end user authentication page for MDM macos setup.

* Added new host detail query to verify MDM profiles and updated API to include verified status.

* Added documentation in the guide for `fleetctl get mdm-commands`.

* Moved post-DEP (automatic) MDM enrollment to a worker job for increased resiliency with retries.

* Added better UI error for manual enroll MDM modal.

* Updated `GET /api/_version_/fleet/config` to now omits fields `smtp_settings` and `sso_settings` if not set.

* Added a response payload to the `POST /api/latest/fleet/spec/teams` contributor API endpoint so that it returns an object with a `team_ids_by_name` key which maps team names with their corresponding id.

* Ensure we send post-enrollment commands to MDM devices that are re-enrolling after being wiped.

* Added error message to UI when Redis disconnects during a live query session.

* Optimized query used for listing activities on the dashboard.

* Added ability for users to delete multiple pages of hosts.

* Added ability to deselect label filter on host table.

* Added support for value `null` on `FLEET_JIT_USER_ROLE_GLOBAL` and `FLEET_JIT_USER_ROLE_TEAM_*` SAML attributes. Fleet will accept and ignore such `null` attributes.

* Deprecate `enable_jit_role_sync` setting and only change role for existing users if role attributes are set in the `SAMLResponse`.

* Improved styling in sandbox mode.

* Patched a potential security issue.

* Improved icon clarity.

* Fixed issues with the MDM migration flow.

* Fixed a bug with applying team specs via `fleetctl apply` and updating a team via the `PATCH /api/latest/fleet/mdm/teams/{id}` endpoint so that the MDM updates settings (`minimum_version` and `deadline`) are not cleared if not provided in the payload.

* Fixed table formatting for the output of `fleetctl get mdm-command-results`.

* Fixed the `/api/latest/fleet/mdm/apple_bm` endpoint so that it returns 400 instead of 500 when it fails to authenticate with Apple's Business Manager API, as this indicates a Fleet configuration issue with the Apple BM certificate or token.

* Fixed a bug that would show MDM URLs for the same server as different servers if they contain query parameters.

* Fixed an issue preventing a user with the `gitops` role from applying some MDM settings via `fleetctl apply` (the `macos_setup_assistant` and `bootstrap_package` settings).

* Fixed `GET /api/v1/fleet/spec/labels/{name}` endpoint so that it now includes the label id.

* Fixed Observer/Observer+ role being able to see team secrets.

* Fixed UI bug where `inherited_page=0` was incorrectly added to some URLs.

* Fixed misaligned icons in UI.

* Fixed tab misalignment caused by new font.

* Fixed dashed line styling on multiline activities.

* Fixed a bug in the users table where users that are observer+ for all of more than one team were listed as "Various roles".

* Fixed 500 error being returned if SSO session is not found.

* Fixed issue with `chrome_extensions` virtual table not returning a path value on `fleetd-chrome`, which was breaking software ingestion.

* Fixed bug with page navigation inside 'My Device' page.

* Fixed a styling bug in the add hosts modal in sandbox mode.

## Fleet 4.32.0 (May 24, 2023)

* Added support to add a EULA as part of the AEP/DEP unboxing flow.

* DEP enrollments configured with SSO now pre-populate the username/fullname fields during account
  creation.

* Integrated the macOS setup assistant feature with Apple DEP so that the setup assistants are assigned to the enrolled devices.

* Re-assign and update the macOS setup assistants (and the default one) whenever required, such as
  when it is modified, when a host is transferred, a team is deleted, etc.

* Added device-authenticated endpoint to signal the Fleet server to send a webhook request with the
  device UUID and serial number to the webhook URL configured for MDM migration.

* Added UI for new automatic enrollment under the integration settings.

* Added UI for end-user migration setup.

* Changed macOS settings UI to always show the profile status aggregate data.

* Revised validation errors returned for `fleetctl mdm run-command`.

* Added `mdm.macos_migration` to app config.

* Added `PATCH /mdm/apple/setup` endpoint.

* Added `enable_end_user_authentication` to `mdm.macos_setup` in global app config and team config
  objects.

* Now tries to infer the bootstrap package name from the URL on upload if a content-disposition header is not provided.

* Added wildcards to host search so when searching for different accented characters you get more results.

* Can now reorder (and bookmark) policy tables by failing count.

* On the login and password reset pages, added email validation and fixed some minor styling bugs.

* Ensure sentence casing on labels on host details page.

* Fix 3 Windows CIS benchmark policies that had false positive results initally merged March 24.

* Fix of Fleet Server returning a duplicate OS version for Windows.

* Improved loading UI for disk encryption controls page.

* The 'GET /api/v1/fleet/hosts/{id}' and 'GET /api/v1/fleet/hosts/identifier/{identifier}' now
  include the software installed path on their payload.

* Third party vulnerability integrations now include the installed path of the vulnerable software
  on each host.

* Greyed out unusable select all queries checkbox.

* Added page header for macOS updates UI.

* Back to queries button returns to previous table state.

* Bookmarkable URLs are now source of truth for Manage Queries page table state.

* Added mechanism to refetch MDM enrollment status of a host pending unenrollment (due to a migration to Fleet) at a high interval.

* Made sure every modal in the UI conforms to a consistent system of widths.

* Team admins and team maintainers cannot save/update a global policy so hide the save button when viewing or running a global policy.

* Policy description has text area instead of one-line area.

* Users can now see the filepath of software on a host.

* Added version info metadata file to Windows installer.

* Fixed a bug where policy automations couldn't be updated without a webhook URL.

* Fixed tooltip misalignment on software page.

## Fleet 4.31.1 (May 10, 2023)

* Fixed a bug that prevented bootstrap packages and the `fleetd` agent from being installed when the server had a database replica configured.

## Fleet 4.31.0 (May 1, 2023)

* Added `gitops` user role to Fleet. GitOps users are users that can manage configuration.

* Added the `fleetctl get mdm-commands` command to get a list of MDM commands that were executed. Added the `GET /api/latest/fleet/mdm/apple/commands` API endpoint.

* Added Fleet UI flows for uploading, downloading, deleting, and viewing information about a Fleet MDM
  bootstrap package.

* Added `apple_bm_enabled_and_configured` to app config responses.

* Added support for the `mdm.macos_setup.macos_setup_assistant` key in the 'config' and 'team' YAML
  payloads supported by `fleetctl apply`.

* Added the endpoints to set, get and delete the macOS setup assistant associated with a team or no team (`GET`, `POST` and `DELETE` methods on the `/api/latest/fleet/mdm/apple/enrollment_profile` path).

* Added functionality to gate Apple MDM login behind SAML authentication.

* Added new "verifying" status for MDM profiles.

* Migrated MDM status values from "applied" to "verifying" and updated associated endpoints.

* Updated macOS settings status filters and aggregate counts to more accurately reflect the status of
FileVault settings.

* Filter out non-`observer_can_run` queries for observers in `fleetctl get queries` to match the UI behavior.

* Fall back to a previous NVD release if the asset we want is not in the latest release.

* Users can now click back to software to return to the filtered host details software tab or filtered manage software page.

* Users can now bookmark software table filters.

* Added a maximum height to the teams dropdown, allowing the user to scroll through a large number of
  teams.

* Present the 403 error page when a user with no access logs in.

* Back to hosts and back to software in host details and software details return to previous table
  state.

* Bookmarkable URLs are now the source of truth for Manage Host and Manage Software table states.

* Removed old Okta configuration that was only documented for internal usage. These configs are being replaced for a general approach to gate profiles behind SSO.

* Removed any host's packs information for observers and observer plus in UI.

* Added `changed_macos_setup_assistant` and `deleted_macos_setup_assistant` activities for the macOS setup assistant setting.

* Hide reset sessions in user dropdown for current user.

* Added a suite of UI logic for premium features in the Sandbox environment.

* In Sandbox, added "Premium Feature" icons for premium-only option to designate a policy as "Critical," as well
  as copy to the tooltip above the icon next to policies designated "Critical" in the Manage policies table.

* Added a star to let a sandbox user know that the "Probability of exploit" column of the Manage
  Software page is a premium feature.

* Added "Premium Feature" icons for premium-only columns of the Vulnerabilities table when in
Sandbox mode.

* Inform prospective customers that Teams is a Premium feature.

* Fixed animation for opening edit user modal.

* Fixed nav bar buttons not responsively resizing when small screen widths cannot fit default size nav bar.

* Fixed a bug with and improved the overall experience of tabbed navigation through the setup flow.

* Fixed `/api/_version/fleet/logout` to return HTTP 401 if unauthorized.

* Fixed endpoint to return proper status code (401) on `/api/fleet/orbit/enroll` if secret is invalid.

* Fixed a bug where a white bar appears at the top of the login page before the app renders.

* Fixed bug in manage hosts table where UI elements related to row selection were displayed to a team
  observer user when that user was also a team and maintainer or admin on another team.

* Fixed bug in add policy UI where a user that is team maintainer or team admin cannot access the UI
  to save a new policy if that user is also an observer on another team.

* Fixed UI bug where dashboard links to hosts filtered by platform did not carry over the selected
  team filter.

* Fixed not showing software card on dashboard when clicking on vulnerabilities.

* Fixed a UI bug where fields on the "My account" page were cut off at smaller viewport widths.

* Fixed software table to match UI spec (responsively hidden vulnerabilities/probability of export column under 990px width).

* Fixed a bug where bundle information displayed in tooltips over a software's name was mistakenly
  hidden.

* Fixed an HTTP 500 on `GET /api/_version_/fleet/hosts` returned when `mdm_enrollment_status` is invalid.

## Fleet 4.30.1 (Apr 12, 2023)

* Fixed a UI bug introduced in version 4.30 where the "Show inherited policies" button was not being
  displayed.

* Fixed inherited schedules not appearing on page reload. 

## Fleet 4.30.0 (Apr 10, 2023)

* Removed both `FLEET_MDM_APPLE_ENABLE` and `FLEET_DEV_MDM_ENABLED` feature flags.

* Automatically send a configuration profile for the `fleetd` agent to teams that use DEP enrollment.

* DEP JSON profiles are now automatically created with default values when the server is run.

* Added the `--mdm` and `--mdm-pending` flags to the `fleetctl get hosts` command to list hosts enrolled in Fleet MDM and pending enrollment in Fleet MDM, respectively.

* Added support for the "enrolled" value for the `mdm_enrollment_status` filter and the new `mdm_name` filter for the "List hosts", "Count hosts" and "List hosts in label" endpoints.

* Added the `fleetctl mdm run-command` command, to run any of the [Apple-supported MDM commands](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) on a host.

* Added the `fleetctl get mdm-command-results` sub-command to get the results for a previously-executed MDM command.

* Added API support to filter the host by the disk encryption status on "GET /hosts", "GET /hosts/count",  and "GET /labels/:id/hosts" endpoints.

* Added API endpoint for disk encryption aggregate status data.

* Automatically install `fleetd` for DEP enrolled hosts.

* Updated hosts' profiles status sync to set to "pending" immediately after an action that affects their list of profiles.

* Updated FileVault configuration profile to disallow device user from disabling full-disk encryption. 

* Updated MDM settings so that they are consistent, and updated documentation for clarity, completeness and correctness.

* Added `observer_plus` user role to Fleet. Observers+ are observers that can run any live query.

* Added a premium-only "Published" column to the vulnerabilities table to display when a vulnerability was first published.

* Improved version detection for macOS apps. This fixes some false positives in macOS vulnerability detection.

* If a new CPE translation rule is pushed, the data in the database should reflect that.

* If a false positive is patched, the data in the database should reflect that.

* Include the published date from NVD in the vulnerability object in the API and the vulnerability webhooks (premium feature only).

* User management table informs which users only have API access.

* Added configuration option `websockets_allow_unsafe_origin` to optionally disable the websocket origin check.

* Added new config `prometheus.basic_auth.disable` to allow running the Prometheus endpoint without HTTP Basic Auth.

* Added missing tables to be cleared on host deletion (those that reference the host by UUID instead of ID).

* Introduced new email backend capable of sending email directly using SES APIs.

* Upgraded Go version to 1.19.8 (includes minor security fixes for HTTP DoS issues).

* Uninstalling applications from hosts will remove the corresponding entry in `software` if no more hosts have the application installed.

* Removed the unused "Issuer URI" field from the single sign-on configuration page of the UI.

* Fixed an issue where some icons would appear clipped at certain zoom levels.

* Fixed a bug where some empty table cells were slightly different colors.

* Fixed e-mail sending on user invites and user e-mail change when SMTP server has credentials.

* Fixed logo misalignment.

* Fixed a bug where for certain org logos, the user could still click on it even outside the navbar.

* Fixed styling bugs on the SelectQueryModal.

* Fixed an issue where custom org logos might be displayed off-center.

* Fixed a UI bug where in certain states, there would be extra space at the right edge of the Manage Hosts table.

## Fleet 4.29.1 (Mar 31, 2023)

* Fixed a migration that was causing `fleet prepare db` to fail due to changes in the collation of the tables. IMPORTANT: please make sure to have a database backup before running migrations.

* Fixed an issue where users would see the incorrect disk encryption banners on the My Device page.

## Fleet 4.29.0 (Mar 22, 2023)

* Added implementation of Fleetd for Chrome.

* Added the `mdm.macos_settings.enable_disk_encryption` option to the `fleetctl apply` configuration
  files of "config" and "team" kind as a Fleet Premium feature.

* Added `mdm.macos_settings.disk_encryption` and `mdm.macos_settings.action_required` status fields in the response for a single host (`GET /hosts/{id}` and `GET /device/{token}` endpoints).

* Added MDM solution name to `host.mdm`in API responses.

* Added support for fleetd to enroll a device using its serial number (in addition to its system
  UUID) to help avoid host-matching issues when a host is first created in Fleet via the MDM
  automatic enrollment (Apple Business Manager).

* Added ability to filter data under the Hosts tab by the aggregate status of hosts' MDM-managed macos
settings.

* Added activity feed items for enabling and disabling disk encryption with MDM.

* Added FileVault banners on the Host Details and My Device pages.

* Added activities for when macOS disk encryption setting is enabled or disabled.

* Added UI for fleet mdm managed disk encryption toggling and the disk encryption aggregate data.

* Added support to update a team's disk encryption via the Modify Team (`PATCH /api/latest/fleet/teams/{id}`) endpoint.

* Added a new API endpoint to gate access to an enrollment profile behind Okta authentication.

* Added new configuration values to integrate Okta in the DEP MDM flow.

* Added `GET /mdm/apple/profiles/summary` endpoint.

* Updated API endpoints that use `team_id` query parameter so that `team_id=0`
  filters results to include only hosts that are not assigned to any team.

* Adjusted the `aggregated_stats` table to compute and store statistics for "no team" in addition to
  per-team and for all teams.

* Added MDM profiles status filter to hosts endpoints.

* Added indicators of aggregate host count for each possible status of MDM-enforced mac settings
  (hidden until 4.30.0).

* As part of JIT provisioning, read user roles from SAML custom attributes.

* Added Win 10 policies for CIS Benchmark 18.x.

* Added Win 10 policies for CIS Benchmark 2.3.17.x.

* Added Win 10 policies for CIS Benchmark 2.3.10.x.

* Documented CIS Windows10 Benchmarks 9.2.x to cis policy queries.

* Document CIS Windows10 Benchmarks 9.3.x to cis policy queries.

* Added button to show query on policy results page.

* Run periodic cleanup of pending `cron_stats` outside the `schedule` package to prevent Fleet outages from breaking cron jobs.

* Added an invitation for users to upgrade to Premium when viewing the Premium-only "macOS updates"
  feature.

* Added an icon on the policy table to indicate if a policy is marked critical.

* Added `"instanceID"` (aka `owner` of `locks`) to `schedule` logging (to help troubleshooting when
  running multiple Fleet instances).

* Introduce UUIDs to Fleet errors and logs.

* Added EndeavourOS, Manjaro, openSUSE Leap and Tumbleweed to HostLinuxOSs.

* Global observer can view settings for all teams.

* Team observers can view the team's settings.

* Updated translation rules so that Docker Desktop can be mapped to the correct CPE.

* Pinned Docker image hashes in Dockerfiles for increased security.

* Remove the `ATTACH` check on SQL osquery queries (osquery bug fixed a while ago in 4.6.0).

* Don't return internal error information on Fleet API requests (internal errors are logged to stderr).

* Fixed an issue when applying the configuration YAML returned by `fleetctl get config` with
  `fleetctl apply` when MDM is not enabled.

* Fixed a bug where `fleetctl trigger` doesn't release the schedule lock when the triggered run
  spans the regularly scheduled interval.

* Fixed a bug that prevented starting the Fleet server with MDM features if Apple Business Manager
  (ABM) was not configured.

* Fixed incorrect MDM-related settings documentation and payload response examples.

* Fixed bug to keep team when clicking on policy tab twice.

* Fixed software table links that were cutting off tooltip.

* Fixed authorization action used on host/search endpoint.

## Fleet 4.28.1 (March 14, 2023) 

* Fixed a bug that prevented starting the Fleet server with MDM features if Apple Business Manager (ABM) was not configured.

## Fleet 4.28.0 (Feb 24, 2023)

* Added logic to ingest and decrypt FileVault recovery keys on macOS if Fleet's MDM is enabled.

* Create activity feed types for the creation, update, and deletion of macOS profiles (settings) via
  MDM.

* Added an API endpoint to retrieve a host disk encryption key for macOS if Fleet's MDM is enabled.

* Added UI implementation for users to upload, download, and deleted macos profiles.

* Added activity feed types for the creation, update, and deletion of macOS profiles (settings) via
  MDM.

* Added API endpoints to create, delete, list, and download MDM configuration profiles.

* Added "edited macos profiles" activity when updating a team's (or no team's) custom macOS settings via `fleetctl apply`.

* Enabled installation and auto-updates of Nudge via Orbit.

* Added support for providing `macos_settings.custom_settings` profiles for team (with Fleet Premium) and no-team levels via `fleetctl apply`.

* Added `--policies-team` flag to `fleetctl apply` to easily import a group of policies into a team.

* Remove requirement for Rosetta in installation of macOS packages on Apple Silicon. The binaries have been "universal" for a while now, but the installer still required Rosetta until now.

* Added max height on org logo image to ensure consistent height of the nav bar.

* UI default policies pre-select targeted platform(s) only.

* Parse the Mac Office release notes and use that for doing vulnerability processing.

* Only set public IPs on the `host.public_ip` field and add documentation on how to properly configure the deployment to ingest correct public IPs from enrolled devices.

* Added tooltip with link to UI when Public IP address cannot be determined.

* Update to better URL validation in UI.

* Set policy platforms using the platform checkboxes as a user would expect the options to successfully save.

* Standardized on a default value for empty cells in the UI.

* Added link to query table in UI source (fleetdm.com/tables/table_name).

* Added live query distributed interval warnings on select targets picker and live query result page.

* Added a macOS settings indicator and modal on the host details and device user pages.

* Added configuration parameters for the filesystem logging destination -- max_size, max_age, and max_backups are now configurable rather than hardcoded values.

* Live query/policy selecting "All hosts" is mutually exclusive from other filters.

* Minor server changes to support Fleetd for ChromeOS (to be released soon).

* Fixed `network_interface_unix` and `network_interface_windows` to ingest "Private IPs" only
  (filter out "Public IPs").

* Fixed how the Fleet MDM server URL is generated when stored for hosts enrolled in Fleet MDM.

* Fixed a panic when loading information for a host enrolled in MDM and its `is_server` field is
  `NULL`.

* Fixed bug with host count on hosts filtered by operating system version.

* Fixed permissions warnings reported by Suspicious Package in macos pkg installers. These warnings
  appeared to be purely cosmetic.

* Fixed UI bug: Long words in activity feed wrap within the div.

## Fleet 4.27.1 (Feb 16, 2023)

* Fixed "Turn off MDM" button appearing on host details without Fleet MDM enabled. 

* Upgrade Go to 1.19.6 to remediate some low severity [denial of service vulnerabilities](https://groups.google.com/g/golang-announce/c/V0aBFqaFs_E/m/CnYKgKwBBQAJ) in the standard library.

## Fleet 4.27.0 (Feb 3, 2023)

* Added API endpoint to unenroll a host from Fleet's MDM.

* Added Request CSR and Change default MDM BM team modals to Integrations > MDM.

* Added a `notifications` object to the response payload of `GET /api/fleet/orbit/config` that includes a `renew_enrollment_profile` field to indicate to fleetd that it needs to run a command on the device to renew the DEP enrollment profile.

* Added modal for automatic enrollment of a macOS host to MDM.

* Integrated with CSR request endpoint in fleet UI.

* Updated `Select targets` UI so that `Platforms`, `Teams`, and `Labels` become `AND` filters. Selecting 2 or more `Platforms`, `Teams`, and `Labels` continue to behave as `OR` filters.

* Added new activities to the activities API when a host is enrolled/unenrolled from Fleet's MDM.

* Implemented macOS update version content panel.

* Added an activity `edited_macos_min_version` when the required minimum macOS version is updated.

* Added the `GET /device/{token}/mdm/apple/manual_enrollment_profile` endpoint to allow downloading the manual MDM enrollment profile from the "My Device" page in Fleet Desktop.

* Run authorization checks before processing policy specs.

* Implemented the new Controls page and updated styling of the site-level navigation.

* Made `fleetctl get teams --yaml` output compatible with `fleetctl apply -f`.

* Added the `POST /api/v1/fleet/mdm/apple/request_csr` endpoint to trigger a Certificate Signing Request to fleetdm.com and return the associated APNs private key and SCEP certificate and key.

* Added mdm enrollment status and mdm server url to `GET /hosts` and `GET /hosts/:id` endpoint
  responses.

* Added keys to the `GET /config` and `GET /device/:token` endpoints to inform if Fleet's MDM is properly configured.

* Add edited min macos version activity.

* User can hover over host UUID to see and copy full ID string.

* Made the 'Back to all hosts' link on the host details page fall back to the default path to the
  manage hosts page. This addresses a bug in this functionality when the user navigates directly
  with the URL.

* Implemented the ability for an authorized user to unenroll a host from MDM on its host details page. The host must be enrolled in MDM and online.

* Added nixos to the list of platforms that are detected at linux distributions.

* Allow to configure a minimum macOS version and a deadline for hosts enrolled into Fleet's MDM.

* Added license expiry to account information page for premium users.

* Removed stale time from loading team policies/policy automation so users are provided accurate team data when toggling between teams.

* Updated to software empty states and host details empty states.

* Changed default hosts per page from 100 to 50.

* Support `CrOS` as a valid platform string for customers with ChromeOS hosts.

* Clean tables at smaller screen widths.

* Log failed login attempts for user+pw and SSO logins (in the activity feed).

* Added `meta` attribute to `GET /activities` endpoint that includes pagination metadata. Fixed edge case
  on UI for pagination buttons on activities card.

* Fleet Premium shows pending hosts on the dashboard and manage host page.

* Use stricter file permissions in `fleetctl updates add` command.

* When table only has 1 host, remove bulky tooltip overflow.

* Documented the Apple Push Notification service (APNs) and Apple Business Manager (ABM) setup and renewal steps.

* Fixed pagination on manage host page.


## Fleet 4.26.0 (Jan 13, 2023)

* Added functionality to ingest device information from Apple MDM endpoints so that an MDM device can
  be surfaced in Fleet results while the initial enrollment of the device is pending.

* Added new activities to the activities API when a host is enrolled/unenrolled from Fleet's MDM.

* Added option to filter hosts by MDM enrollment status "pending" to surface devices ordered through
  Apple Business Manager that are still pending enrollment in Fleet's MDM.

* Added a flag to indicate if the Apple Business Manager terms and conditions have changed and must
  be accepted to have automatic enrollment of hosts work again. A banner is added to the output of
  `fleetctl` commands when this is the case.

* Added side navigation layout to the integration page and conditionally show MDM section.

* Added application configuration: mdm.apple_bm_default_team.

* Added modal to allow user to download an enrollment profile required for Fleet MDM enrollment.

* Added new configuration option to set default team for Apple Business Manager.

* Added a software_updated_at column denoting when software was updated for a host.

* Generate audit log for activities (supported log plugins are: `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`).

* Added locally-formated datetime tooltips.

* Updated software empty states.

* Autofocus first entry of all forms for better UX.

* Added pendo to sandbox instances.

* Added bookmarkability of url when it includes the `query` query param on the manage hosts page.

* Pack target details show on right side of dropdown.

* Updated buttons to the the new style guide.

* Added a way to override a detail query or disable it through app config.

* Invalid query string will not result in neverending spinner.

* Fixed an issue causing enrollment profiles to fail if the server URL had a trailing slash.

* Fixed an issue that made the first SCEP enrollment during the MDM check-in flow fail in a new setup.

* Fixed panic in `/api/{version}/fleet/hosts/{d}/mdm` when the host does not have MDM data.

* Fixed ingestion of MDM data with empty server URLs (meaning the host is not enrolled to an MDM server).

* Removed stale time from loading team policies/policy automation so users are provided accurate team data when toggling between teams.

## Fleet 4.25.0 (Dec 22, 2022)

* Added new activity that records create/edit/delete user roles.

* Log all successful logins as activity and all attempts with ip in stderr.

* Added API endpoint to generate DEP public and private keys.

* Added ability to mark policy as critical with Fleet Premium.

* Added ability to mark policies run automation for all already failing hosts.

* Added `fleet serve` configuration flags for Apple Push Notification service (APNs) and Simple
  Certificate Enrollment Protocol (SCEP) certificates and keys.

* Added `fleet serve` configuration flags for Apple Business Manager (BM).

* Added `fleetctl trigger` command to trigger an ad hoc run of all jobs in a specified cron
  schedule.

* Added the `fleetctl get mdm_apple` command to retrieve the Apple MDM configuration information. MDM features are not ready for production and are currently in development. These features are disabled by default.

* Added the `fleetctl get mdm_apple_bm` command to retrieve the Apple Business Manager configuration information.

* Added `fleetctl` command to generate APNs CSR and SCEP CA certificate and key pair.

* Add `fleetctl` command to generate DEP public and private keys.

* Windows installer now ensures that the installed osquery version gets removed before installing Orbit.

* Build on Ubuntu 20 to resolve glibc changes that were causing issues for older Docker runtimes.

* During deleting host flow, inform users how to prevent re-enrolling hosts.

* Added functionality to report if a carve failed along with its error message.

* Added the `redis.username` configuration option for setups that use Redis ACLs.

* Windows installer now ensures that no files are left on the filesystem when orbit uninstallation
  process is kicked off.

* Improve how we are logging failed detail queries and windows os version queries.

* Spiffier UI: Add scroll shadows to indicate horizontal scrolling to user.

* Add counts_update_at attribute to GET /hosts/summary/mdm response. update GET /labels/:id/hosts to
  filter by mdm_id and mdm_enrollment_status query params. add mobile_device_management_solution to
  response from GET /labels/:id/hosts when including mdm_id query param. add mdm information to UI for
  windows/all dashboard and host details.

* Fixed `fleetctl query` to use custom HTTP headers if configured.

* Fixed how we are querying and ingesting disk encryption in linux to workaround an osquery bug.

* Fixed buggy input field alignments.

* Fixed to multiselect styling.

* Fixed bug where manually triggering a cron run that preempts a regularly scheduled run causes
  an unexpected shift in the start time of the next interval.

* Fixed an issue where the height of the label for some input fields changed when an error message is displayed.

* Fixed the alignment of the "copy" and "show" button icons in the manage enroll secrets and get API
  token modals.

## Fleet 4.24.1 (Dec 7, 2022)

**This is a security release.**

* Update Go to 1.19.4

## Fleet 4.24.0 (Dec 1, 2022)

* Improve live query activity item in the activity feed on the Dashboard page. Each item will include the user’s name, as well as an option to show the query. If the query has been saved, the item will include the query’s name.

* Improve navigation on Host details page and Dashboard page by adding the ability to navigate back to a tab (ex. Policies) and filter (ex. macOS) respectively.

* Improved performance of the Fleet server by decreasing CPU usage by 20% and memory usage by 3% on average.

* Added tooltips and updated dropdown choices on Hosts and Host details pages to clarify the meanings of "Status: Online" and "Status: Offline."

* Added “Void Linux” to the list of recognized distributions.

* Added clickable rows to software tables to view all hosts filtered by software.

* Added support for more OS-specific osquery command-line flags in the agent options.

* Added links to evented tables and columns that require user context in the query side panel.

* Improved CPU and memory usage of Fleet.

* Removed the Preview payload button from the usage statistics page, as well as its associated logic and unique styles. [See the example usage statistics payload](https://fleetdm.com/docs/using-fleet/usage-statistics#what-is-included-in-usage-statistics-in-fleet) in the Using Fleet documentation.

* Removed tooltips and conditional coloring in the disk space graph for Linux hosts.

* Reduced false negatives for the query used to determine encryption status on Linux systems.

* Fixed long software name from aligning centered.

* Fixed a discrepancy in the height of input labels when there’s a validation error.

## Fleet 4.23.0 (Nov 14, 2022)

* Added preview screenshots for Jira and Zendesk vulnerability tickets for Premium users.

* Improve host detail query to populate primary ip and mac address on host.

* Add option to show public IP address in Hosts table.

* Improve ingress resource by replacing the template with a most recent version, that enables:
  - Not having any annotation hardcoded, all annotations are optional.
  - Custom path, as of now it was hardcoded to `/*`, but depending on the ingress controller, it can require an extra annotation to work with regular expressions.
  - Specify ingressClassName, as it was hardcoded to `gce`, and this is a setting that might be different on each cluster.

* Added ingestion of host orbit version from `orbit_info` osquery extension table.

* Added number of hosts enrolled by orbit version to usage statistics payload.

* Added number of hosts enrolled by osquery version to usage statistics payload.

* Added arch and linuxmint to list of linux distros so that their data is displayed and host count includes them.

* When submitting invalid agent options, inform user how to override agent options using fleetctl force flag.

* Exclude Windows Servers from mdm lists and aggregated data.

* Activity feed includes editing team config file using fleetctl.

* Update Go to 1.19.3.

* Host details page includes information about the host's disk encryption.

* Information surfaced to device user includes all summary/about information surfaced in host details page.

* Support low_disk_space filter for endpoint /labels/{id}/hosts.

* Select targets pages implements cleaner icons.

* Added validation of unknown keys for the Apply Teams Spec request payload (`POST /spec/teams` endpoint).

* Orbit MSI installer now includes the necessary manifest file to use windows_event_log as a logger_plugin.

* UI allows for filtering low disk space hosts by platform.

* Add passed policies column on the inherited policies table for teams.

* Use the MSRC security bulletins to scan for Windows vulnerabilities. Detected vulnerabilities are inserted in a new table, 'operating_system_vulnerabilities'.

* Added vulnerability scores to Jira and Zendesk integrations for Fleet Premium users.

* Improve database usage to prevent some deadlocks.

* Added ingestion of disk encryption status for hosts, and added that flag in the response of the `GET /hosts/{id}` API endpoint.

* Trying to add a host with 0 enroll secrets directs user to manage enroll secrets.

* Detect Windows MDM solutions and add mdm endpoints.

* Styling updates on login and forgot password pages.

* Add UI polish and style fixes for query pages.

* Update styling of tooltips and modals.

* Update colors, issues icon.

* Cleanup dashboard styling.

* Add tooling for writing integration tests on the frontend.

* Fixed host details page so munki card only shows for mac hosts.

* Fixed a bug where duplicate vulnerability webhook requests, jira, and zendesk tickets were being
  made when scanning for vulnerabilities. This affected ubuntu and redhat hosts that support OVAL
  vulnerability detection.

* Fixed bug where password reset token expiration was not enforced.

* Fixed a bug in `fleetctl apply` for teams, where a missing `agent_options` key in the YAML spec
  file would clear the existing agent options for the team (now it leaves it unchanged). If the key
  is present but empty, then it clears the agent options.

* Fixed bug with our CPE matching process. UTM.app was matching to the wrong CPE.

* Fixed an issue where fleet would send invalid usage stats if no hosts were enrolled.

* Fixed an Orbit MSI installer bug that caused Orbit files not to be removed during uninstallation.

## Fleet 4.22.1 (Oct 27, 2022)

* Fixed the error response of the `/device/:token/desktop` endpoint causing problems on free Fleet Desktop instances on versions `1.3.x`.

## Fleet 4.22.0 (Oct 20, 2022)

* Added usage statistics for the weekly count of aggregate policy violation days. One policy violation day is counted for each policy that a host is failing, measured as of the time the count increments. The count increments once per 24-hour interval and resets each week.

* Fleet Premium: Add ability to see how many and which hosts have low disk space (less than 32GB available) on the **Home** page.

* Fleet Premium: Add ability to see how many and which hosts are missing (offline for at least 30 days) on the **Home** page.

* Improved the query console by indicating which columns are required in the WHERE clause, indicated which columns are platform-specific, and adding example queries for almost all osquery tables in the right sidebar. These improvements are also live on [fleetdm.com/tables](https://fleetdm.com/tables)

* Added a new display name for hosts in the Fleet UI. To determine the display name, Fleet uses the `computer_name` column in the [`system_info` table](https://fleetdm.com/tables/system_info). If `computer_name` isn't present, the `hostname` is used instead.

* Added functionality to consider device tokens as expired after one hour. This change is not compatible with older versions of Fleet Desktop. We recommend to manually update Orbit and Fleet Desktop to > v1.0.0 in addition to upgrading the server if:
  * You're managing your own TUF server.
  * You have auto-updates disabled (`fleetctl package [...] --disable-updates`)
  * You have channels pinned to an older version (`fleetctl package [...] --orbit-channel 1.0.0 --desktop-channel 1.1.0`).

* Added security headers to HTML, CSV, and installer responses.

* Added validation of the `command_line_flags` object in the Agent Options section of Organization Settings and Team Settings.

* Added logic to clean up irrelevant policies for a host on re-enrollment (e.g., if a host changes its OS from linux to macOS or it changes teams).

* Added the `inherited_policies` array to the `GET /teams/{team_id}/policies` endpoint that lists the global policies inherited by the team, along with the pass/fail counts for the hosts on that team.

* Added a new UI state for when results are coming in from a live query or policy query.

* Added better team name suggestions to the Create teams modal.

* Clarified last seen time and last fetched time in the Fleet UI.

* Translated technical error messages returned by Agent options validation to be more user-friendly.

* Renamed machine serial to serial number and IPv4 properly to private IP address.

* Fleet Premium: Updated Fleet Desktop to use the `/device/{token}/desktop` API route to display the number of failing policies.

* Made host details software tables more responsive by adding links to software details.

* Fixed a bug in which a user would not be rerouted to the Home page if already logged in.

* Fixed a bug in which clicking the select all checkbox did not select all in some cases.

* Fixed a bug introduced in 4.21.0 where a Windows-specific query was being sent to non-Windows hosts, causing an error in query ingestion for `directIngestOSWindows`.

* Fixed a bug in which uninstalled software (DEB packages) appeared in Fleet.

* Fixed a bug in which a team that didn't have `config.features` settings was edited via the UI, then both `features.enable_host_users` and `features.enable_software_inventory` would be false instead of the global default.

* Fixed a bug that resulted in false negatives for vulnerable versions of Zoom, Google Chrome, Adobe Photoshop, Node.js, Visual Studio Code, Adobe Media Encoder, VirtualBox, Adobe Premiere Pro, Pip, and Firefox software.

* Fixed bug that caused duplicated vulnerabilities to be sent to third-party integrations.

* Fixed panic in `ingestKubequeryInfo` query ingestion.

* Fixed a bug in which `host_count` and `user_count` returned as `0` in the `teams/{id}` endpoint.

* Fixed a bug in which tooltips for Munki issue would be cut off at the edge of the browser window.

* Fixed a bug in which tooltips for Munki issue would be cut off at the edge of the browser window.

* Fixed a bug in which running `fleetctl apply` with the `--dry-run` flag would fail in some cases.

* Fixed a bug in which **Hosts** table displayed 20 hosts per page.

* Fixed a server panic that occured when a team was edited via YAML without an `agent_options` key.

* Fixed an bug where Pop!\_OS hosts were not being included in the linux hosts count on the hosts dashboard page.


## Fleet 4.21.0 (Sep 28, 2022)

* Fleet Premium: Added the ability to know how many hosts and which hosts, on a team, are failing a global policy.

* Added validation to the `config` and `teams` configuration files. Fleet can be managed with [configuration files (YAML syntax)](https://fleetdm.com/docs/using-fleet/configuration-files) and the fleetctl command line tool. 

* Added the ability to manage osquery flags remotely. This requires [Orbit, Fleet's agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager). If at some point you revoked an old enroll secret, this feature won't work for hosts that were added to Fleet using this old enroll secret. To manage osquery flags on these hosts, we recommend deploying a new package. Check out the instructions [here on GitHub](https://github.com/fleetdm/fleet/issues/7377).

* Added a `/api/v1/fleet/device/{token}/desktop` API route that returns only the number of failing policies for a specific host.

* Added support for kubequery.

* Added support for an `AC_TEAM_ID` environment variable when creating [signed installers for macOS hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#signing-fleetd-installers).

* Made cards on the **Home** page clickable.

* Added es_process_file_events, password_policy, and windows_update_history tables to osquery.

* Added activity items to capture when, and by who, agent options are edited.

* Added logging to capture the user’s email upon successful login.

* Increased the size of placeholder text from extra small to small.

* Fixed an error that cleared the form when adding a new integration.

* Fixed an error generating Windows packages with the fleetctl package on non-English localizations of Windows.

* Fixed a bug that showed the small screen overlay when trying to print.

* Fixed the UI bug that caused the label filter dropdown to go under the table header.

* Fixed side panel tooltips to not be wider than side panel causing scroll bug.


## Fleet 4.20.1 (Sep 15, 2022)

**This is a security release.**

* **Security**: Upgrade Go to 1.19.1 to resolve a possible HTTP denial of service vulnerability ([CVE-2022-27664](https://nvd.nist.gov/vuln/detail/CVE-2022-27664)).

* Fixed a bug in which [vulnerability automations](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations) sent duplicate webhooks.

* Fixed a bug in which logging in with single sign-on (SSO) did not work after a failed authorization attempt.

* Fixed a migration error. This only affects Fleet instances that use MariaDB. MariaDB is not [officially supported](https://fleetdm.com/docs/deploying/faq#what-mysql-versions-are-supported). Future issues specific to MariaDB may not be fixed quickly (or at all). We strongly advise migrating to MySQL 8.0.19+.

* Fixed a bug on the **Edit pack** page in which no targets are shown in the target picker.

* Fixed a styling bug on the **Host details > Query > Select a query** modal.

## Fleet 4.20.0 (Sep 9, 2022)

* Add ability to know how many hosts, and which hosts, have Munki issues. This information is presented on the **Home > macOS** page and **Host details** page. This information is also available in the [`GET /api/v1/fleet/macadmins`](https://fleetdm.com/docs/using-fleet/rest-api#get-aggregated-hosts-mobile-device-management-mdm-and-munki-information) and [`GET /api/v1/fleet/hosts/{id}/macadmins`](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-mobile-device-management-mdm-and-munki-information) and API routes.

* Fleet Premium: Added ability to test features, like software inventory, on canary teams by adding a [`features` section](https://fleetdm.com/docs/using-fleet/configuration-files#features) to the `teams` YAML document.

* Improved vulnerability detection for macOS hosts by improving detection of Zoom, Ruby, and Node.js vulnerabilities. Warning: For users that download and sync Fleet's vulnerability feeds manually, there are [required adjustments](https://github.com/fleetdm/fleet/issues/6628) or else vulnerability processing will stop working. Users with the default vulnerability processing settings can safely upgrade without adjustments.

* Fleet Premium: Improved the vulnerability automations by adding vulnerability scores (EPSS probability, CVSS scores, and CISA-known exploits) to the webhook payload. Read more about vulnerability automations on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations).

* Renamed the `host_settings` section to `features` in the the [`config` YAML file](https://fleetdm.com/docs/using-fleet/configuration-files#features). But `host_settings` is still supported for backwards compatibility.

* Improved the activity feed by adding the ability to see who modified agent options and when modifications occurred. This information is available on the Home page in the Fleet UI and the [`GET /activites` API route](https://fleetdm.com/docs/using-fleet/rest-api#activities).

* Improved the [`config` YAML documentation](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings).

* Improved the **Hosts** page for smaller screen widths.

* Improved the building of osquery installers for Windows (`.msi` packages).

* Added a **Show query** button on the **Schedule** page, which adds the ability to quickly see a query's SQL.

* Improved the Fleet UI by adding loading spinners to all buttons that create or update entities in Fleet (e.g., users).

* Fixed a bug in which a user could not reach some teams in the UI via pagination if there were more than 20 teams.

* Fixed a bug in which a user could not reach some users in the UI via pagination if there were more than 20 users.

* Fixed a bug in which duplicate vulnerabilities (CVEs) sometimes appeared on **Software details** page.

* Fixed a bug in which the count in the **Issues** column (exclamation tooltip) in the **Hosts** table would sometimes not appear.

* Fixed a bug in which no error message would appear if there was an issue while setting up Fleet.

* Fixed a bug in which no error message would appear if users were creating or editing a label with a name or description that was too long.

* Fixed a big in which the example payload for usage statistics included incorrect key names.

* Fixed a bug in which the count above the **Software** table would sometimes not appear.

* Fixed a bug in which the **Add hosts** button would not be displayed when search returned 0 hosts.

* Fixed a bug in which modifying filters on the **Hosts** page would not return the user to the first page of the **Hosts** table. 

## Fleet 4.19.1 (Sep 1, 2022)

* Fix a migration error that may occur when upgrading to Fleet 4.19.0.

* Fix a bug in which the incorrect operating system was displayed for Windows hosts on the **Hosts** page and **Host details** page.

## Fleet 4.19.0 (Aug 22, 2022)

* Warning: Please upgrade to 4.19.1 instead of 4.19.0 due to a migration error included in 4.19.0. Like all releases, Fleet 4.19.1 includes all changes included in 4.19.0.

* Fleet Premium: De-anonymize usage statistics by adding an `organization` property to the usage statistics payload. For Fleet Free instances, organization is reported as "unknown". Documentation on how to disable usage statistics, can be found [here on fleetdm.com](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics).

* Fleet Premium: Added support for Just-in-time (JIT) user provisioning via SSO. This adds the ability to
automatically create Fleet user accounts when a new users attempts to log in to Fleet via SSO. New
Fleet accounts are given the [Observer role](https://fleetdm.com/docs/using-fleet/permissions#user-permissions).

* Improved performance for aggregating software inventory. Aggregate software inventory is displayed on the **Software page** in the Fleet UI.

* Added the ability to see the vendor for Windows programs in software inventory. Vendor data is available in the [`GET /software` API route](https://fleetdm.com/docs/using-fleet/rest-api#software).

* Added a **Mobile device management (MDM) solutions** table to the **Home > macOS** page. This table allows users to see a list of all MDM solutions their hosts are enrolled to and drill down to see which hosts are enrolled to each solution. Note that MDM solutions data is updated as hosts send fresh osquery results to Fleet. This typically occurs in an hour or so of upgrading.

* Added a **Operating systems** table to the **Home > Windows** page. This table allows users to see a list of all Windows operating systems (ex. Windows 10 Pro 21H2) their hosts are running and drill down to see which hosts are running which version. Note that Windows operating system data is updated as hosts send fresh osquery results to Fleet. This typically occurs in an hour or so of upgrading.

* Added a message in `fleetctl` to that notifies users to run `fleet prepare` instead of `fleetctl prepare` when running database migrations for Fleet.

* Improved the Fleet UI by maintaining applied, host filters when a user navigates back to the Hosts page from an
individual host's **Host details** page.

* Improved the Fleet UI by adding consistent styling for **Cancel** buttons.

* Improved the **Queries**, **Schedule**, and **Policies** pages in the Fleet UI by page size to 20
  items. 

* Improve the Fleet UI by informing the user that Fleet only supports screen widths above 768px.

* Added support for asynchronous saving of the hosts' scheduled query statistics. This is an
experimental feature and should only be used if you're seeing performance issues. Documentation
for this feature can be found [here on fleetdm.com](https://fleetdm.com/docs/deploying/configuration#osquery-enable-async-host-processing).

* Fixed a bug in which the **Operating system** and **Munki versions** cards on the **Home > macOS**
page would not stack vertically at smaller screen widths.

* Fixed a bug in which multiple Fleet Desktop icons would appear on macOS computers.

* Fixed a bug that prevented Windows (`.msi`) installers from being generated on Windows machines.

## Fleet 4.18.0 (Aug 1, 2022)

* Added a Call to Action to the failing policy banner in Fleet Desktop. This empowers end-users to manage their device's compliance. 

* Introduced rate limiting for device authorized endpoints to improve the security of Fleet Desktop. 

* Improved styling for tooltips, dropdowns, copied text, checkboxes and buttons. 

* Fixed a bug in the Fleet UI causing text to be truncated in tables. 

* Fixed a bug affecting software vulnerabilities count in Host Details.

* Fixed "Select Targets" search box and updated to reflect currently supported search values: hostname, UUID, serial number, or IPv4.

* Improved disk space reporting in Host Details. 

* Updated frequency formatting for Packs to match Schedules. 

* Replaced "hosts" count with "results" count for live queries.

* Replaced "Uptime" with "Last restarted" column in Host Details.

* Removed vulnerabilities that do not correspond to a CVE in Fleet UI and API.

* Added standard password requirements when users are created by an admin.

* Updated the regexp we use for detecting the major/minor version on OS platforms.

* Improved calculation of battery health based on cycle count. “Normal” corresponds to cycle count < 1000 and “Replacement recommended” corresponds to cycle count >= 1000.

* Fixed an issue with double quotes usage in SQL query, caused by enabling `ANSI_QUOTES` in MySQL.

* Added automated tests for Fleet upgrades.

## Fleet 4.17.1 (Jul 22, 2022)

* Fixed a bug causing an error when converting users to SSO login. 

* Fixed a bug causing the **Edit User** modal to hang when editing multiple users.

* Fixed a bug that caused Ubuntu hosts to display an inaccurate OS version. 

* Fixed a bug affecting exporting live query results.

* Fixed a bug in the Fleet UI affecting live query result counts.

* Improved **Battery Health** processing to better reflect the health of batteries for M1 Macs.

## Fleet 4.17.0 (Jul 8, 2022)

* Added the number of hosts enrolled by operating system (OS) and its version to usage statistics. Also added the weekly active users count to usage statistics.
Documentation on how to disable usage statistics, can be found [here on fleetdm.com](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics).

* Fleet Premium and Fleet Free: Fleet desktop is officially out of beta. This application shows users exactly what's going on with their device and gives them the tools they need to make sure it is secure and aligned with policies. They just need to click an icon in their menu bar. 

* Fleet Premium and Fleet Free: Fleet's osquery installer is officially out of beta. Orbit is a lightweight wrapper for osquery that allows you to easily deploy, configure and keep osquery up-to-date across your organization. 

* Added native support for M1 Macs.

* Added battery health tracking to **Host details** page.

* Improved reporting of error states on the health dashboard and added separate health checks for MySQL and Redis with `/healthz?check=mysql` and `/healthz?check=redis`.

* Improved SSO login failure messaging.

* Fixed osquery tables that report incorrect platforms.

* Added `docker_container_envs` table to the osquery table schema on the **Query* page.

* Updated Fleet host detail query so that the `os_version` for Ubuntu hosts reflects the accurate patch number.

* Improved accuracy of `software_host_counts` by removing hosts from the count if any software has been uninstalled.

* Improved accuracy of the `last_restarted` date. 

* Fixed `/api/_version_/fleet/hosts/identifier/{identifier}` to return the correct value for `host.status`.

* Improved logging when fleetctl encounters permissions errors.

* Added support for scanning RHEL-based and Fedora hosts for vulnerable software using OVAL definitions.

* Fixed SQL generated for operating system version policies to reduce false negatives.

## Fleet 4.16.0 (Jun 20, 2022)

* Fleet Premium: Added the ability to set a Custom URL for the "Transparency" link included in Fleet Desktop. This allows you to use custom branding, as well as gives you control over what information you want to share with your end-users. 

* Fleet Premium: Added scoring to vulnerability detection, including EPSS probability score, CVSS base score, and known exploits. This helps you to quickly categorize which threats need attention today, next week, next month, or "someday."

* Added a ticket-workflow for policy automations. Configured Fleet to automatically create a Jira issue or Zendesk ticket when one or more hosts fail a specific policy.

* Added [Open Vulnerability and Assement Language](https://access.redhat.com/solutions/4161) (`OVAL`) processing for Ubuntu hosts. This increases the accuracy of detected vulnerabilities. 

* Added software details page to the Fleet UI.

* Improved live query experience by saving the state of selected targets and adding count of visible results when filtering columns.

* Fixed an issue where the **Device user** page redirected to login if an expired session token was present. 

* Fixed an issue that caused a delay in availability of **My device** in Fleet Desktop.

* Added support for custom headers for requests made to `fleet` instances by the `fleetctl` command.

* Updated to an improved `users` query in every query we send to osquery.

* Fixed `no such table` errors for `mdm` and `munki_info` for vanilla osquery MacOS hosts.

* Fixed data inconsistencies in policy counts caused when a host was re-enrolled without a team or in a different one.

* Fixed a bug affecting `fleetctl debug` `archive` and `errors` commands on Windows.

* Added `/api/_version_/fleet/device/{token}/policies` to retrieve policies for a specific device. This endpoint can only be accessed with a premium license.

* Added `POST /targets/search` and `POST /targets/count` API endpoints.

* Updated `GET /software`, `GET /software/{:id}`, and `GET /software/count` endpoints to no include software that has been removed from hosts, but not cleaned up yet (orphaned).

## Fleet 4.15.0 (May 26, 2022)

* Expanded beta support for vulnerability reporting to include both Zendesk and Jira integration. This allows users to configure Fleet to automatically create a Zendesk ticket or Jira issue when a new vulnerability (CVE) is detected on your hosts.

* Expanded beta support for Fleet Desktop to Mac and Windows hosts. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a host, generate a Fleet-osquery installer with `fleetctl package` and include the `--fleet-desktop` flag. Then, open this installer on the device.

* Added the ability to see when software was last used on Mac hosts in the **Host Details** view in the Fleet UI. Allows you to know how recently an application was accessed and is especially useful when making decisions about whether to continue subscriptions for paid software and distributing licensces. 

* Improved security by increasing the minimum password length requirement for Fleet users to 12 characters.

* Added Policies tab to **Host Details** page for Fleet Premium users.

* Added `device_mapping` to host information in UI and API responses.

* Deprecated "MIA" host status in UI and API responses.

* Added CVE scores to `/software` API endpoint responses when available.

* Added `all_linux_count` and `builtin_labels` to `GET /host_summary` response.

* Added "Bundle identifier" information as tooltip for macOS applications on Software page.

* Fixed an issue with detecting root directory when using `orbit shell`.

* Fixed an issue with duplicated hosts being sent in the vulnerability webhook payload.

* Added the ability to select columns when exporting hosts to CSV.

* Improved the output of `fleetclt debug errors` and added the ability to print the errors to stdout via the `-stdout` flag.

* Added support for Docker Compose V2 to `fleetctl preview`.

* Added experimental option to save responses to `host_last_seen` queries to the database in batches as well as the ability to configure `enable_async_host_processing` settings for `host_last_seen`, `label_membership` and `policy_membership` independently. 
* Expanded `wifi_networks` table to include more data on macOS and fixed compatibility issues with newer MacOS releases.

* Improved precision in unseen hosts reports sent by the host status webhook.

* Increased MySQL `group_concat_max_len` setting from default 1024 to 4194304.

* Added validation for pack scheduled query interval.

* Fixed instructions for enrolling hosts using osqueryd.

## Fleet 4.14.0 (May 9, 2022)

* Added beta support for Jira integration. This allows users to configure Fleet to
  automatically create a Jira issue when a new vulnerability (CVE) is detected on
  your hosts.

* Added a "Show query" button on the live query results page. This allows users to double-check the
  syntax used and compare this to their results without leaving the current view.

* Added a [Postman
  Collection](https://www.postman.com/fleetdm/workspace/fleet/collection/18010889-c5604fe6-7f6c-44bf-a60c-46650d358dde?ctx=documentation)
  for the Fleet API. This allows users to easily interact with Fleet's API routes so that they can
  build and test integrations.

* Added beta support for Fleet Desktop on Linux. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a Linux device, first add the
`--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
includes Fleet Desktop. Then, open this installer on the device.

* Added `last_opened_at` property, for macOS software, to the **Host details** API route (`GET /hosts/{id}`).

* Improved the **Settings** pages in the the Fleet UI.

* Improved error message retuned when running `fleetctl query` command with missing or misspelled hosts.

* Improved the empty states and forms on the **Policies** page, **Queries** page, and **Host details** page in the Fleet UI.

* All duration settings returned by `fleetctl get config --include-server-config` were changed from
nanoseconds to an easy to read format.

* Fixed a bug in which the "Bundle identifier" tooltips displayed on **Host details > Software** did not
  render correctly.

* Fixed a bug in which the Fleet UI would render an empty Google Chrome profiles on the **Host details** page.

* Fixed a bug in which the Fleet UI would error when entering the "@" characters in the **Search targets** field.

* Fixed a bug in which a scheduled query would display the incorrect name when editing the query on
  the **Schedule** page.

* Fixed a bug in which a deprecation warning would be displayed when generating a `deb` or `rpm`
  Fleet-osquery package when running the `fleetctl package` command.

* Fixed a bug that caused panic errors when running the `fleet serve --debug` command.

## Fleet 4.13.2 (Apr 25, 2022)

* Fixed a bug with os versions not being updated. Affected deployments using MySQL < 5.7.22 or equivalent AWS RDS Aurora < 2.10.1.

## Fleet 4.13.1 (Apr 20, 2022)

* Fixed an SSO login issue introduced in 4.13.0.

* Fixed authorization errors encountered on the frontend login and live query pages.

## Fleet 4.13.0 (Apr 18, 2022)

### This is a security release.

* **Security**: Fixed several post-authentication authorization issues. Only Fleet Premium users that
  have team users are affected. Fleet Free users do not have access to the teams feature and are
  unaffected. See the following security advisory for details: https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr

* Improved performance of software inventory on Windows hosts.

* Added `basic​_auth.username` and `basic_auth.password` [Prometheus configuration options](https://fleetdm.com/docs/deploying/configuration#prometheus). The `GET
/metrics` API route is now disabled if these configuration options are left unspecified. 

* Fleet Premium: Add ability to specify a team specific "Destination URL" for policy automations.
This allows the user to configure Fleet to send a webhook request to a unique location for
policies that belong to a specific team. Documentation on what data is included the webhook
request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations)

* Added the ability to see the total number of hosts with a specific macOS version (ex. 12.3.1) on the
**Home > macOS** page. This information is also available via the [`GET /os_versions` API route](https://fleetdm.com/docs/using-fleet/rest-api#get-host-os-versions).

* Added the ability to sort live query results in the Fleet UI.

* Added a "Vulnerabilities" column to **Host details > Software** page. This allows the user see and search for specific vulnerabilities (CVEs) detected on a specific host.

* Updated vulnerability automations to fire anytime a vulnerability (CVE), that is detected on a
  host, was published to the
  National Vulnerability Database (NVD) in the last 30 days, is detected on a host. In previous
  versions of Fleet, vulnerability automations would fire anytime a CVE was published to NVD in the
  last 2 days.

* Updated the **Policies** page to ask the user to wait to see accurate passing and failing counts for new and recently edited policies.

* Improved API-only (integration) users by removing the requirement to reset these users' passwords
  before use. Documentation on how to use API-only users can be found here on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user).

* Improved the responsiveness of the Fleet UI by adding tablet screen width support for the **Software**,
  **Queries**, **Schedule**, **Policies**, **Host details**, **Settings > Teams**, and **Settings > Users** pages.

* Added Beta support for integrating with Jira to automatically create a Jira issue when a
  new vulnerability (CVE) is detected on a host in Fleet. 

* Added Beta support for Fleet Desktop on Windows. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a Windows device, first add the
`--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
includes Fleet Desktop. Then, open this installer on the device.

* Fixed a bug in which downloading [Fleet's vulnerability database](https://github.com/fleetdm/nvd) failed if the destination directory specified
was not in the `tmp/` directory.

* Fixed a bug in which the "Updated at" time was not being updated for the "Mobile device management
(MDM) enrollment" and "Munki versions" information on the **Home > macOS** page.

* Fixed a bug in which Fleet would consider Docker network interfaces to be a host's primary IP address.

* Fixed a bug in which tables in the Fleet UI would present misaligned buttons.

* Fixed a bug in which Fleet failed to connect to Redis in standalone mode.
## Fleet 4.12.1 (Apr 4, 2022)

* Fixed a bug in which a user could not log in with basic authentication. This only affects Fleet deployments that use a [MySQL read replica](https://fleetdm.com/docs/deploying/configuration#mysql).

## Fleet 4.12.0 (Mar 24, 2022)

* Added ability to update which platform (macOS, Windows, Linux) a policy is checked on.

* Added ability to detect compatibility for custom policies.

* Increased the default session duration to 5 days. Session duration can be updated using the
  [`session_duration` configuration option](https://fleetdm.com/docs/deploying/configuration#session-duration).

* Added ability to see the percentage of hosts that responded to a live query.

* Added ability for user's with [admin permissions](https://fleetdm.com/docs/using-fleet/permissions#user-permissions) to update any user's password.

* Added [`content_type_value` Kafka REST Proxy configuration
  option](https://fleetdm.com/docs/deploying/configuration#kafkarest-content-type-value) to allow
  the use of different versions of the Kafka REST Proxy.

* Added [`database_path` GeoIP configuration option](https://fleetdm.com/docs/deploying/configuration#database-path) to specify a GeoIP database. When configured,
  geolocation information is presented on the **Host details** page and in the `GET /hosts/{id}` API route.

* Added ability to retrieve a host's public IP address. This information is available on the **Host
  details** page and `GET /hosts/{id}` API route.

* Added instructions and materials needed to add hosts to Fleet using [plain osquery](https://fleetdm.com/docs/using-fleet/adding-hosts#plain-osquery). These instructions
can be found in **Hosts > Add hosts > Advanced** in the Fleet UI.

* Added Beta support for Fleet Desktop on macOS. Fleet Desktop allows the device user to see
  information about their device. To add Fleet Desktop to a macOS device, first add the
  `--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
  includes Fleet Desktop. Then, open this installer on the device.

* Reduced the noise of osquery status logs by only running a host vital query, which populate the
**Host details** page, when the query includes tables that are compatible with a specific host.

* Fixed a bug on the **Edit pack** page in which the "Select targets" element would display the hover effect for the wrong target.

* Fixed a bug on the **Software** page in which software items from deleted hosts were not removed.

* Fixed a bug in which the platform for Amazon Linux 2 hosts would be displayed incorrectly.

## Fleet 4.11.0 (Mar 7, 2022)

* Improved vulnerability processing to reduce the number of false positives for RPM packages on Linux hosts.

* Fleet Premium: Added a `teams` key to the `packs` yaml document to allow adding teams as targets when using CI/CD to manage query packs.

* Fleet premium: Added the ability to retrieve configuration for a specific team with the `fleetctl get team --name
<team-name-here>` command.

* Removed the expiration for API tokens for API-only users. API-only users can be created using the
  `fleetctl user create --api-only` command.

* Improved performance of the osquery query used to collect software inventory for Linux hosts.

* Updated the activity feed on the **Home page** to include add, edit, and delete policy activities.
  Activity information is also available in the `GET /activities` API route.

* Updated Kinesis logging plugin to append newline character to raw message bytes to properly format NDJSON for downstream consumers.

* Clarified why the "Performance impact" for some queries is displayed as "Undetermined" in the Fleet
  UI.

* Added instructions for using plain osquery to add hosts to Fleet in the Fleet View these instructions by heading to **Hosts > Add hosts > Advanced**.

* Fixed a bug in which uninstalling Munki from one or more hosts would result in inaccurate Munki
  versions displayed on the **Home > macOS** page.

* Fixed a bug in which a user, with access limited to one or more teams, was able to run a live query
against hosts in any team. This bug is not exposed in the Fleet UI and is limited to users of the
`POST run` API route. 

* Fixed a bug in the Fleet UI in which the "Select targets" search bar would not return the expected hosts.

* Fixed a bug in which global agent options were not updated correctly when editing these options in
the Fleet UI.

* Fixed a bug in which the Fleet UI would incorrectly tag some URLs as invalid.

* Fixed a bug in which the Fleet UI would attempt to connect to an SMTP server when SMTP was disabled.

* Fixed a bug on the Software page in which the "Hosts" column was not filtered by team.

* Fixed a bug in which global maintainers were unable to add and edit policies that belonged to a
  specific team.

* Fixed a bug in which the operating system version for some Linux distributions would not be
displayed properly.

* Fixed a bug in which configuring an identity provider name to a value shorter than 4 characters was
not allowed.

* Fixed a bug in which the avatar would not appear in the top navigation.


## Fleet 4.10.0 (Feb 13, 2022)

* Upgraded Go to 1.17.7 with security fixes for crypto/elliptic (CVE-2022-23806), math/big (CVE-2022-23772), and cmd/go (CVE-2022-23773). These are not likely to be high impact in Fleet deployments, but we are upgrading in an abundance of caution.

* Added aggregate software and vulnerability information on the new **Software** page.

* Added ability to see how many hosts have a specific vulnerable software installed on the
  **Software** page. This information is also available in the `GET /api/v1/fleet/software` API route.

* Added ability to send a webhook request if a new vulnerability (CVE) is
found on at least one host. Documentation on what data is included the webhook
request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations).

* Added aggregate Mobile Device Management and Munki data on the **Home** page.

* Added email and URL validation across the entire Fleet UI.

* Added ability to filter software by "Vulnerable" on the **Host details** page.

* Updated standard policy templates to use new naming convention. For example, "Is FileVault enabled on macOS
devices?" is now "Full disk encryption enabled (macOS)."

* Added db-innodb-status and db-process-list to `fleetctl debug` command.

* Fleet Premium: Added the ability to generate a Fleet installer and manage enroll secrets on the **Team details**
  page. 

* Added the ability for users with the observer role to view which platforms (macOS, Windows, Linux) a query
  is compatible with.

* Improved the experience for editing queries and policies in the Fleet UI.

* Improved vulnerability processing for NPM packages.

* Added supports triggering a webhook for newly detected vulnerabilities with a list of affected hosts.

* Added filter software by CVE.

* Added the ability to disable scheduled query performance statistics.

* Added the ability to filter the host summary information by platform (macOS, Windows, Linux) on the **Home** page.

* Fixed a bug in Fleet installers for Linux in which a computer restart would stop the host from
  reporting to Fleet.

* Made sure ApplyTeamSpec only works with premium deployments.

* Disabled MDM, Munki, and Chrome profile queries on unsupported platforms to reduce log noise.

* Properly handled paths in CVE URL prefix.

## Fleet 4.9.1 (Feb 2, 2022)

### This is a security release.

* **Security**: Fixed a vulnerability in Fleet's SSO implementation that could allow a malicious or compromised SAML Service Provider (SP) to log into Fleet as an existing Fleet user. See https://github.com/fleetdm/fleet/security/advisories/GHSA-ch68-7cf4-35vr for details.

* Allowed MSI packages generated by `fleetctl package` to reinstall on Windows without uninstall.

* Fixed a bug in which a team's scheduled queries didn't render correctly on the **Schedule** page.

* Fixed a bug in which a new policy would always get added to "All teams" rather than the selected team.

## Fleet 4.9.0 (Jan 21, 2022)

* Added ability to apply a `policy` yaml document so that GitOps workflows can be used to create and
  modify policies.

* Added ability to run a live query that returns 1,000+ results in the Fleet UI by adding
  client-side pagination to the results table.

* Improved the accuracy of query platform compatibility detection by adding recognition for queries
  with the `WITH` expression.

* Added ability to open a page in the Fleet UI in a new tab by "right-clicking" an item in the navigation.

* Improved the [live query API route (`GET /api/v1/queries/run`)](https://fleetdm.com/docs/using-fleet/rest-api#run-live-query) so that it successfully return results for Fleet
  instances using a load balancer by reducing the wait period to 25 seconds.

* Improved performance of the Fleet UI by updating loading states and reducing the number of requests
  made to the Fleet API.

* Improved performance of the MySQL database by updating the queries used to populate host vitals and
  caching the results.

* Added [`read_timeout` Redis configuration
  option](https://fleetdm.com/docs/deploying/configuration#redis-read-timeout) to customize the
  maximum amount of time Fleet should wait to receive a response from a Redis server.

* Added [`write_timeout` Redis configuration
  option](https://fleetdm.com/docs/deploying/configuration#redis-write-timeout) to customize the
  maximum amount of time Fleet should wait to send a command to a Redis server.

* Fixed a bug in which browser extensions (Google Chrome, Firefox, and Safari) were not included in
  software inventory.

* Improved the security of the **Organization settings** page by preventing the browser from requesting
  to save SMTP credentials.

* Fixed a bug in which an existing pack's targets were not cleaned up after deleting hosts, labels, and teams.

* Fixed a bug in which non-existent queries and policies would not return a 404 not found response.

### Performance

* Our testing demonstrated an increase in max devices served in our load test infrastructure to 70,000 from 60,000 in v4.8.0.

#### Load Test Infrastructure

* Fleet server
  * AWS Fargate
  * 2 tasks with 1024 CPU units and 2048 MiB of RAM.

* MySQL
  * Amazon RDS
  * db.r5.2xlarge

* Redis
  * Amazon ElastiCache 
  * cache.m5.large with 2 replicas (no cluster mode)

#### What was changed to accomplish these improvements?

* Optimized the updating and fetching of host data to only send and receive the bare minimum data
  needed. 

* Reduced the number of times host information is updated by caching more data.

* Updated cleanup jobs and deletion logic.

#### Future improvements

* At maximum DB utilization, we found that some hosts fail to respond to live queries. Future releases of Fleet will improve upon this.

## Fleet 4.8.0 (Dec 31, 2021)

* Added ability to configure Fleet to send a webhook request with all hosts that failed a
  policy. Documentation on what data is included the webhook
  request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations).

* Added ability to find a user's device in Fleet by filtering hosts by email associated with a Google Chrome
  profile. Requires the [macadmins osquery
  extension](https://github.com/macadmins/osquery-extension) which comes bundled in [Fleet's osquery
  installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). 
  
* Added ability to see a host's Google Chrome profile information using the [`GET
  api/v1/fleet/hosts/{id}/device_mapping` API
  route](https://fleetdm.com/docs/using-fleet/rest-api#get-host-device-mapping).

* Added ability to see a host's mobile device management (MDM) enrollment status, MDM server URL,
  and Munki version on a host's **Host details** page. Requires the [macadmins osquery
  extension](https://github.com/macadmins/osquery-extension) which comes bundled in [Fleet's osquery
  installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). 

* Added ability to see a host's MDM and Munki information with the [`GET
  api/v1/fleet/hosts/{id}/macadmins` API
  route](https://fleetdm.com/docs/using-fleet/rest-api#list-mdm-and-munki-information-if-available).

* Improved the handling of certificates in the `fleetctl package` command by adding a check for a
  valid PEM file.

* Updated [Prometheus Go client library](https://github.com/prometheus/client_golang) which
  results in the following breaking changes to the [`GET /metrics` API
  route](https://fleetdm.com/docs/using-fleet/monitoring-fleet#metrics):
  `http_request_duration_microseconds` is now `http_request_duration_seconds_bucket`,
  `http_request_duration_microseconds_sum` is now `http_request_duration_seconds_sum`,
  `http_request_duration_microseconds_count` is now `http_request_duration_seconds_count`,
  `http_request_size_bytes` is now `http_request_size_bytes_bucket`, and `http_response_size_bytes`
  is now `http_response_size_bytes_bucket`

* Improved performance when searching and sorting hosts in the Fleet UI.

* Improved performance when running a live query feature by reducing the load on Redis.

* Improved performance when viewing software installed across all hosts in the Fleet
  UI.

* Fixed a bug in which the Fleet UI presented the option to download an undefined certificate in the "Generate installer" instructions.

* Fixed a bug in which database migrations failed when using MariaDB due to a migration introduced in Fleet 4.7.0.

* Fixed a bug that prevented hosts from checking in to Fleet when Redis was down.

## Fleet 4.7.0 (Dec 14, 2021)

* Added ability to create, modify, or delete policies in Fleet without modifying saved queries. Fleet
  4.7.0 introduces breaking changes to the `/policies` API routes to separate policies from saved
  queries in Fleet. These changes will not affect any policies previously created or modified in the
  Fleet UI.

* Turned on vulnerability processing for all Fleet instances with software inventory enabled.
  [Vulnerability processing in Fleet](https://fleetdm.com/docs/using-fleet/vulnerability-processing)
  provides the ability to see all hosts with specific vulnerable software installed. 

* Improved the performance of the "Software" table on the **Home** page.

* Improved performance of the MySQL database by changing the way a host's users information   is saved.

* Added ability to select from a library of standard policy templates on the **Policies** page. These
  pre-made policies ask specific "yes" or "no" questions about your hosts. For example, one of
  these policy templates asks "Is Gatekeeper enabled on macOS devices?"

* Added ability to ask whether or not your hosts have a specific operating system installed by
  selecting an operating system policy on the **Host details** page. For example, a host that is
  running macOS 12.0.1 will present a policy that asks "Is macOS 12.0.1 installed on macOS devices?"

* Added ability to specify which platform(s) (macOS, Windows, and/or Linux) a policy is checked on.

* Added ability to generate a report that includes which hosts are answering "Yes" or "No" to a 
  specific policy by running a policy's query as a live query.

* Added ability to see the total number of installed software software items across all your hosts.

* Added ability to see an example scheduled query result that is sent to your configured log
  destination. Select "Schedule a query" > "Preview data" on the **Schedule** page to see the 
  example scheduled query result.

* Improved the host's users information by removing users without login shells and adding users 
  that are not associated with a system group.

* Added ability to see a Fleet instance's missing migrations with the `fleetctl debug migrations`
  command. The `fleet serve` and `fleet prepare db` commands will now fail if any unknown migrations
  are detected.

* Added ability to see syntax errors as your write a query in the Fleet UI.

* Added ability to record a policy's resolution steps that can be referenced when a host answers "No" 
  to this policy.

* Added server request errors to the Fleet server logs to allow for troubleshooting issues with the 
Fleet server in non-debug mode.

* Increased default login session length to 24 hours.

* Fixed a bug in which software inventory and disk space information was not retrieved for Debian hosts.

* Fixed a bug in which searching for targets on the **Edit pack** page negatively impacted performance of 
  the MySQL database.

* Fixed a bug in which some Fleet migrations were incompatible with MySQL 8.

* Fixed a bug that prevented the creation of osquery installers for Windows (.msi) when a non-default 
  update channel is specified.

* Fixed a bug in which the "Software" table on the home page did not correctly filtering when a
  specific team was selected on the **Home** page.

* Fixed a bug in which users with "No access" in Fleet were presented with a perpetual 
  loading state in the Fleet UI.

## Fleet 4.6.2 (Nov 30, 2021)

* Improved performance of the **Home** page by removing total hosts count from the "Software" table.

* Improved performance of the **Queries** page by adding pagination to the list of queries.

* Fixed a bug in which the "Shell" column of the "Users" table on the **Host details** page would sometimes fail to update.

* Fixed a bug in which a host's status could quickly alternate between "Online" and "Offline" by increasing the grace period for host status.

* Fixed a bug in which some hosts would have a missing `host_seen_times` entry.

* Added an `after` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow for cursor pagination.

* Added a `disable_failing_policies` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow the API request to respond faster if failing policies count information is not needed.

## Fleet 4.6.1 (Nov 21, 2021)

* Fixed a bug (introduced in 4.6.0) in which Fleet used progressively more CPU on Redis, resulting in API and UI slowdowns and inconsistency.

* Made `fleetctl apply` fail when the configuration contains invalid fields.

## Fleet 4.6.0 (Nov 18, 2021)

* Fleet Premium: Added ability to filter aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page. The aggregate host data is also available in the [`GET /host_summary API route`](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-summary).

* Fleet Premium: Added ability to move pending invited users between teams.

* Fleet Premium: Added `fleetctl updates rotate` command for rotation of keys in the updates system. The `fleetctl updates` command provides the ability to [self-manage an agent update server](https://fleetdm.com/docs/deploying/fleetctl-agent-updates).

* Enabled the software inventory by default for new Fleet instances. The software inventory feature can be turned on or off using the [`enable_software_inventory` configuration option](https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration).

* Updated the JSON payload for the host status webhook by renaming the `"message"` property to `"text"` so that the payload can be received and displayed in Slack.

* Removed the deprecated `app_configs` table from Fleet's MySQL database. The `app_config_json` table has replaced it.

* Improved performance of the policies feature for Fleet instances with over 100,000 hosts.

* Added instructions in the Fleet UI for generating an osquery installer for macOS, Linux, or Windows. Documentation for generating an osquery installer and distributing the installer to your hosts to add them to Fleet can be found here on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/adding-hosts)

* Added ability to see all the software, and filter by vulnerable software, installed across all your hosts on the **Home** page. Each software's `name`, `version`, `hosts_count`, `vulnerabilities`, and more is also available in the [`GET /software` API route](https://fleetdm.com/docs/using-fleet/rest-api#software) and `fleetctl get software` command.

* Added ability to add, edit, and delete enroll secrets on the **Hosts** page.

* Added ability to see aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page.

* Added ability to see all of the queries scheduled to run on a specific host on the **Host details** page immediately after a query is added to a schedule or pack.

* Added a "Shell" column to the "Users" table on the **Host details** page so users can now be filtered to see only those who have logged in.

* Packaged osquery's `certs.pem` in `fleetctl package` to improve TLS compatibility.

* Added support for packaging an osquery flagfile with `fleetctl package --osquery-flagfile`.

* Used "Fleet osquery" rather than "Orbit osquery" in packages generated by `fleetctl package`.

* Clarified that a policy in Fleet is a yes or no question you can ask about your hosts by replacing "Passing" and "Failing" text with "Yes" and "No" respectively on the **Policies** page and **Host details** page.

* Added ability to see the original author of a query on the **Query** page.

* Improved the UI for the "Software" table and "Policies" table on the **Host details** page so that it's easier to pivot to see all hosts with a specific software installed or answering "No" to a specific policy.

* Fixed a bug in which modifying a specific target for a live query, in target selector UI, would deselect a different target.

* Fixed a bug in which the user was navigated to a non existent page, in the Fleet UI, after saving a pack.

* Fixed a bug in which long software names in the "Software" table caused the bundle identifier tooltip to be inaccessible.

## Fleet 4.5.1 (Nov 10, 2021)

* Fixed performance issues with search filtering on manage queries page.

* Improved correctness and UX for query platform compatibility.

* Fleet Premium: Shows correct hosts when a team is selected.

* Fixed a bug preventing login for new SSO users.

* Added always return the `disabled` value in the `GET /api/v1/fleet/packs/{id}` API (previously it was
  sometimes left out).

## Fleet 4.5.0 (Nov 1, 2021)

* Fleet Premium: Added a Team admin user role. This allows users to delegate the responsibility of managing team members in Fleet. Documentation for the permissions associated with the Team admin and other user roles can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions).

* Added Apache Kafka logging plugin. Documentation for configuring Kafka as a logging plugin can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#kafka-rest-proxy-logging). Thank you to Joseph Macaulay for adding this capability.

* Added support for [MinIO](https://min.io/) as a file carving backend. Documentation for configuring MinIO as a file carving backend can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#minio). Thank you to Chandra Majumdar and Ben Edwards for adding this capability.

* Added support for generating a `.pkg` osquery installer on Linux without dependencies (beyond Docker) with the `fleetctl package` command.

* Improved the performance of vulnerability processing by making the process consume less RAM. Documentation for the vulnerability processing feature can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Added the ability to run a live query and receive results using only the Fleet REST API with a `GET /api/v1/fleet/queries/run` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#run-live-query).

* Added ability to see whether a specific host is "Passing" or "Failing" a policy on the **Host details** page. This information is also exposed in the `GET api/v1/fleet/hosts/{id}` API route. In Fleet, a policy is a "yes" or "no" question you can ask of all your hosts.

* Added the ability to quickly see the total number of "Failing" policies for a particular host on the **Hosts** page with a new "Issues" column. Total "Issues" are also revealed on a specific host's **Host details** page.

* Added the ability to see which platforms (macOS, Windows, Linux) a specific query is compatible with. The compatibility detected by Fleet is estimated based on the osquery tables used in the query.

* Added the ability to see whether your queries have a "Minimal," "Considerable," or "Excessive" performance impact on your hosts. Query performance information is only collected when a query runs as a scheduled query.

  * Running a "Minimal" query, very frequently, has little to no impact on your host's performance.

  * Running a "Considerable" query, frequently, can have a noticeable impact on your host's performance.

  * Running an "Excessive" query, even infrequently, can have a significant impact on your host’s performance.

* Added the ability to see a list of hosts that have a specific software version installed by selecting a software version on a specific host's **Host details** page. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](https://fleetdm.com/docs/deploying/configuration#feature-flags).

* Added the ability to see all vulnerable software detected across all your hosts with the `GET /api/v1/fleet/software` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#software).

* Added the ability to see the exact number of hosts that selected filters on the **Hosts** page. This ability is also available when using the `GET api/v1/fleet/hosts/count` API route.

* Added ability to automatically "Refetch" host vitals for a particular host without manually reloading the page.

* Added ability to connect to Redis with TLS. Documentation for configuring Fleet to use a TLS connection to the Redis server can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-use-tls).

* Added `cluster_read_from_replica` Redis to specify whether or not to prefer readying from a replica when possible. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-read-from-replica).

* Improved experience of the Fleet UI by preventing autocomplete in forms.

* Fixed a bug in which generating an `.msi` osquery installer on Windows would fail with a permission error.

* Fixed a bug in which turning on the host expiry setting did not remove expired hosts from Fleet.

* Fixed a bug in which the Software inventory for some host's was missing `bundle_identifier` information.

## Fleet 4.4.3 (Oct 21, 2021)

* Cached AppConfig in redis to speed up requests and reduce MySQL load.

* Fixed migration compatibility with MySQL GTID replication.

* Improved performance of software listing query.

* Improved MSI generation compatibility (for macOS M1 and some Virtualization configurations) in `fleetctl package`.

## Fleet 4.4.2 (Oct 14, 2021)

* Fixed migration errors under some MySQL configurations due to use of temporary tables.

* Fixed pagination of hosts on host dashboard.

* Optimized HTTP requests on host search.

## Fleet 4.4.1 (Oct 8, 2021)

* Fixed database migrations error when updating from 4.3.2 to 4.4.0. This did not effect upgrades
  between other versions and 4.4.0.

* Improved logging of errors in fleet serve.

## Fleet 4.4.0 (Oct 6, 2021)

* Fleet Premium: Teams Schedules show inherited queries from All teams (global) Schedule.

* Fleet Premium: Team Maintainers can modify and delete queries, and modify the Team Schedule.

* Fleet Premium: Team Maintainers can delete hosts from their teams.

* `fleetctl get hosts` now shows host additional queries if there are any.

* Update default homepage to new dashboard.

* Added ability to bulk delete hosts based on manual selection and applied filters.

* Added display macOS bundle identifiers on software table if available.

* Fixed scroll position when navigating to different pages.

* Fleet Premium: When transferring a host from team to team, clear the Policy results for that host.

* Improved stability of host vitals (fix cases of dropping users table, disk space).

* Improved performance and reliability of Policy database migrations.

* Provided a more clear error when a user tries to delete a query that is set in a Policy.

* Fixed query editor Delete key and horizontal scroll issues.

* Added cleaner buttons and icons on Manage Hosts Page.

## Fleet 4.3.2 (Sept 29, 2021)

* Improved database performance by reducing the amount of MySQL database queries when a host checks in.

* Fixed a bug in which users with the global maintainer role could not edit or save queries. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Fixed a bug in which policies were checked about every second and add a `policy_update_interval` osquery configuration option. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-policy-update-interval).

* Fixed a bug in which edits to a query’s name, description, SQL did not appear until the user refreshed the Edit query page.

* Fixed a bug in which the hosts count for a label returned 0 after modifying a label’s name or description.

* Improved error message when attempting to create or edit a user with an email that already exists.

## Fleet 4.3.1 (Sept 21, 2021)

* Added `fleetctl get software` command to list all software and the detected vulnerabilities. The Vulnerable software feature is currently in Beta. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Added `fleetctl vulnerability-data-stream` command to sync the vulnerabilities processing data streams by hand.

* Added `disable_data_sync` vulnerabilities configuration option to avoid downloading the data streams. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#disable-data-sync).

* Only shows observers the queries they have permissions to run on the **Queries** page. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Added `connect_retry_attempts` Redis configuration option to retry failed connections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-connect-retry-attempts).

* Added `cluster_follow_redirections` Redis configuration option to follow cluster redirections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-follow-redirections).

* Added `max_jitter_percent` osquery configuration option to prevent all hosts from returning data at roughly the same time. Note that this improves the Fleet server performance, but it will now take longer for new labels to populate. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-max-jitter-percent).

* Improved the performance of database migrations.

* Reduced database load for label membership recording.

* Added fail early if the process does not have permissions to write to the logging file.

* Added completely skip trying to save a host's users and software inventory if it's disabled to reduce database load. 

* Fixed a bug in which team maintainers were unable to run live queries against the hosts assigned to their team(s).

* Fixed a bug in which a blank screen would intermittently appear on the **Hosts** page.

* Fixed a bug detecting disk space for hosts.

## Fleet 4.3.0 (Sept 13, 2021)

* Added Policies feature for detecting device compliance with organizational policies.

* Run/edit query experience has been completely redesigned.

* Added support for MySQL read replicas. This allows the Fleet server to scale to more hosts.

* Added configurable webhook to notify when a specified percentage of hosts have been offline for over the specified amount of days.

* Added `fleetctl package` command for building Orbit packages.

* Added enroll secret dialog on host dashboard.

* Exposed free disk space in gigs and percentage for hosts.

* Added 15-minute interval option on Schedule page.

* Cleaned up advanced options UI.

* 404 and 500 page now include buttons for Osquery community Slack and to file an issue

* Updated all empty and error states for cleaner UI.

* Added warning banners in Fleet UI and `fleetctl` for license expiration.

* Rendered query performance information on host vitals page pack section.

* Improved performance for app loading.

* Made team schedule names more user friendly and hide the stats for global and team schedules when showing host pack stats.

* Displayed `query_name` in when referencing scheduled queries for more consistent UI/UX.

* Query action added for observers on host vitals page.

* Added `server_settings.debug_host_ids` to gather more detailed information about what the specified hosts are sending to fleet.

* Allowed deeper linking into the Fleet application by saving filters in URL parameters.

* Renamed Basic Tier to Premium Tier, and Core Tier to Free Tier.

* Improved vulnerability detection compatibility with database configurations.

* MariaDB compatibility fixes: add explicit foreign key constraint and on cascade delete for host_software to allow for hosts with software to be deleted.

* Fixed migration that was incompatible with MySQL primary key requirements (default on DigitalOcean MySQL 5.8).

* Added 30 second SMTP timeout for mail configuration.

* Fixed display of platform Labels on manage hosts page

* Fixed a bug recording scheduled query statistics.

* When a label is removed, ignore query executions for that label.

* Added fleet serve config to change the redis connection timeout and keep alive interval.

* Removed hardcoded limits in label searches when targeting queries.

* Allow host users to be readded.

* Moved email template images from github to fleetdm.com.

* Fixed bug rendering CPU in host vitals.

* Updated the schema for host_users to allow for bulk inserts without locking, and allow for users without unique uid.

* When using dynamic vulnerability processing node, try to create the vulnerability.databases-path.

* Fixed `fleetctl get host <hostname>` to properly output JSON when the command line flag is supplied i.e `fleetctl get host --json foobar`

## Fleet 4.2.4 (Sept 2, 2021)

* Fixed a bug in which live queries would fail for deployments that use Redis Cluster.

* Fixed a bug in which some new Fleet deployments don't include the default global agent options. Documentation for global and team agent options can be found [here](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options).

* Improved how a host's `users` are stored in MySQL to prevent deadlocks. This information is available in the "Users" table on each host's **Host details** page and in the `GET /api/v1/fleet/hosts/{id}` API route.

## Fleet 4.2.3 (Aug 23, 2021)

* Added ability to troubleshoot connection issues with the `fleetctl debug connection` command.

* Improved compatibility with MySQL variants (MariaDB, Aurora, etc.) by removing usage of JSON_ARRAYAGG.

* Fixed bug in which live queries would stop returning results if more than 5 seconds goes by without a result. This bug was introduced in 4.2.1.

* Eliminated double-logging of IP addresses in osquery endpoints.

* Update host details after transferring a host on the details page.

* Logged errors in osquery endpoints to improve debugging.

## Fleet 4.2.2 (Aug 18, 2021)

* Added a new built in label "All Linux" to target all hosts that run any linux flavor.

* Allowed finer grained configuration of the vulnerability processing capabilities.

* Fixed performance issues when updating pack contents.

* Fixed a build issue that caused external network access to panic in certain Linux distros (Ubuntu).

* Fixed rendering of checkboxes in UI when modals appear.

* Orbit: synced critical file writes to disk.

* Added "-o" flag to fleetctl convert command to ensure consistent output rather than relying on shell redirection (this was causing issues with file encodings).

* Fixed table column wrapping for manage queries page.

* Fixed wrapping in Label pills.

* Side panels in UI have a fresher look, Teams/Roles UI greyed out conditionally.

* Improved sorting in UI tables.

* Improved detection of CentOS in label membership.

## Fleet 4.2.1 (Aug 14, 2021)

* Fixed a database issue with MariaDB 10.5.4.

* Displayed updated team name after edit.

* When a connection from a live query websocket is closed, Fleet now timeouts the receive and handles the different cases correctly to not hold the connection to Redis.

* Added read live query results from Redis in a thread safe manner.

* Allows observers and maintainers to refetch a host in a team they belong to.

## Fleet 4.2.0 (Aug 11, 2021)

* Added the ability to simultaneously filter hosts by status (`online`, `offline`, `new`, `mia`) and by label on the **Hosts** page.

* Added the ability to filter hosts by team in the Fleet UI, fleetctl CLI tool, and Fleet API. *Available for Fleet Basic customers*.

* Added the ability to create a Team schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1). *Available for Fleet Basic customers*.

* Added Beta Vulnerable software feature which surfaces vulnerable software on the **Host details** page and the `GET /api/v1/fleet/hosts/{id}` API route. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Added the ability to see which logging destination is configured for Fleet in the Fleet UI. To see this information, head to the **Schedule** page and then select "Schedule a query." Configured logging destination information is also available in the `GET api/v1/fleet/config` API route.

* Improved the `fleetctl preview` experience by downloading Fleet's standard query library and loading the queries into the Fleet UI.

* Improved the user interface for the **Packs** page and **Queries** page in the Fleet UI.

* Added the ability to modify scheduled queries in your Schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1).

* Added the ability to disable the Users feature in Fleet by setting the new `enable_host_users` key to `true` in the `config` yaml, configuration file. For documentation on using configuration files in yaml syntax, check out the [Using yaml files in Fleet](https://fleetdm.com/docs/using-fleet/configuration-files#using-yaml-files-in-fleet) documentation.

* Improved performance of the Software inventory feature. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](https://fleetdm.com/docs/deploying/configuration#feature-flags).

* Improved performance of inserting `pack_stats` in the database. The `pack_stats` information is used to display "Frequency" and "Last run" information for a specific host's scheduled queries. You can find this information on the **Host details** page.

* Improved Fleet server logging so that it is more uniform.

* Fixed a bug in which a user with the Observer role was unable to run a live query.

* Fixed a bug that prevented the new **Home** page from being displayed in some Fleet instances.

* Fixed a bug that prevented accurate sorting issues across multiple pages on the **Hosts** page.

## Fleet 4.1.0 (Jul 26, 2021)

The primary additions in Fleet 4.1.0 are the new Schedule and Activity feed features.

Scheduled lets you add queries which are executed on your devices at regular intervals without having to understand or configure osquery query packs. For experienced Fleet and osquery users, the ability to create new, and modify existing, query packs is still available in the Fleet UI and fleetctl command-line tool. To reach the **Packs** page in the Fleet UI, head to **Schedule > Advanced**.

Activity feed adds the ability to observe when, and by whom, queries are changes, packs are created, live queries are run, and more. The Activity feed feature is located on the new Home page in the Fleet UI. Select the logo in the top right corner of the Fleet UI to navigate to the new **Home** page.

### New features breakdown

* Added ability to create teams and update their respective agent options and enroll secrets using the new `teams` yaml document and fleetctl. Available in Fleet Basic.

* Added a new **Home** page to the Fleet UI. The **Home** page presents a breakdown of the enrolled hosts by operating system.

* Added a "Users" table on the **Host details** page. The `username` information displayed in the "Users" table, as well as the `uid`, `type`, and `groupname` are available in the Fleet REST API via the `/api/v1/fleet/hosts/{id}` API route.

* Added ability to create a user without an invitation. You can now create a new user by heading to **Settings > Users**, selecting "Create user," and then choosing the "Create user" option.

* Added ability to search and sort installed software items in the "Software" table on the **Host details** page. 

* Added ability to delete a user from Fleet using a new `fleetctl user delete` command.

* Added ability to retrieve hosts' `status`, `display_text`, and `labels` using the `fleetctl get hosts` command.

* Added a new `user_roles` yaml document that allows users to manage user roles via fleetctl. Available in Fleet Basic.

* Changed default ordering of the "Hosts" table in the Fleet UI to ascending order (A-Z).

* Improved performance of the Software inventory feature by reducing the amount of inserts and deletes are done in the database when updating each host's
software inventory.

* Removed YUM and APT sources from Software inventory.

* Fixed an issue in which disabling SSO at the organization level would not disable SSO for all users.

* Fixed an issue with data migrations in which enroll secrets are duplicated after the `name` column was removed from the `enroll_secrets` table.

* Fixed an issue in which it was not possible to clear host settings by applying the `config` yaml document. This allows users to successfully remove the `additional_queries` property after adding it.

* Fixed printing of failed record count in AWS Kinesis/Firehose logging plugins.

* Fixed compatibility with GCP Memorystore Redis due to missing CLUSTER command.


## Fleet 4.0.1 (Jul 01, 2021)

* Fixed an issue in which migrations failed on MariaDB MySQL.

* Allowed `http` to be used when configuring `fleetctl` for `localhost`.

* Fixed a bug in which Team information was missing for hosts looked up by Label. 

## Fleet 4.0.0 (Jun 29, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/9-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/10-Teams.md).

### New features breakdown

* Added the ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Added the ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Added the ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Added Redis cluster support. Available in Fleet Core.

* Fixed a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. The following sections call out changes to consider when upgrading to Fleet 4.0.0:

* The structure of Fleet's `.tar.gz` and `.zip` release archives have changed slightly. Deployments that use the binary artifacts may need to update scripts or tooling. The `fleetdm/fleet` Docker container maintains the same API.

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Changeed your SAML SSO URI to use fleet instead of kolide . This is due to the changes to Fleet's API routes outlined in the section above.

* Changeed configuration option `server_tlsprofile` to `server_tls_compatibility`. This options previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity. `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration.

* The `username` artifact has been removed in favor of the more recognizable `name` (Full name). As a result the `email` artifact is now used for uniqueness in Fleet. Upon upgrading to Fleet 4.0.0, existing users will have the `name` field populated with `username`. SAML users may need to update their username mapping to match user emails.

* As of Fleet 4.0.0, Fleet Device Management Inc. periodically collects anonymous information about your instance. Sending usage statistics is turned off by default for users upgrading from a previous version of Fleet. Read more about the exact information collected [here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/11-Usage-statistics.md).

## Fleet 4.0.0 RC3 (Jun 25, 2021)

Primarily teste the new release workflows. Relevant changelog will be updated for Fleet 4.0. 

## Fleet 4.0.0 RC2 (Jun 18, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/8-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/9-Teams.md).

### New features breakdown

* Added the ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Added the ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Added the ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Added Redis cluster support. Available in Fleet Core.

* Fixed a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Changed configuration option `server_tlsprofile` to `server_tls_compatability`. This option previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues


There are currently no known issues in this release. However, we recommend only upgrading to Fleet 4.0.0-rc2 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

## Fleet 4.0.0 RC1 (Jun 10, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to information and features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access. Check out the [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) for a breakdown of the new user roles and their respective capabilities.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily observe and apply operations to consistent groups of hosts. Read more about the Teams feature in [the documentation here](https://fleetdm.com/docs/using-fleet/teams).

There are several known issues that will be fixed for the stable release of Fleet 4.0.0. Therefore, we recommend only upgrading to Fleet 4.0.0 RC1 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

### New features breakdown

* Added the ability to define a user's access to information and features in Fleet by introducing the Admin, Maintainer, and Observer roles.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Added the ability to enroll hosts to one team using team specific enroll secrets.

* Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Used strictly `fleet` in Fleet's configuration, API routes, and environment variables. This means that you must update all usage of `kolide` in these items. The backwards compatibility introduced in Fleet 3.8.0 is no longer valid in Fleet 4.0.0.

* Changed configuration option `server_tlsprofile` to `server_tls_compatability`. This options previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues

* Query packs cannot be targeted to teams.

## Fleet 3.13.0 (Jun 3, 2021)

* Improved performance of the `additional_queries` feature by moving `additional` query results into a separate table in the MySQL database. Please note that the `/api/v1/fleet/hosts` API endpoint now return only the requested `additional` columns. See documentation on the changes to the hosts API endpoint [here](https://github.com/fleetdm/fleet/blob/06b2e564e657492bfbc647e07eb49fd4efca5a03/docs/1-Using-Fleet/3-REST-API.md#list-hosts).

* Fixed a bug in which running a live query in the Fleet UI would return no results and the query would seem "hung" on a small number of devices.

* Improved viewing live query errors in the Fleet UI by including the “Errors” table in the full screen view.

* Improved `fleetctl preview` experience by adding the `fleetctl preview reset` and `fleetctl preview stop` commands to reset and stop simulated hosts running in Docker.

* Added several improvements to the Fleet UI including additional contrast on checkboxes and dropdown pills.

## Fleet 3.12.0 (May 19, 2021)

* Added scheduled queries to the _Host details_ page. Surface the "Name", "Description", "Frequency", and "Last run" information for each query in a pack that apply to a specific host.

* Improved the freshness of host vitals by adding the ability to "refetch" the data on the _Host details_ page.

* Added ability to copy log fields into Google Cloud Pub/Sub attributes. This allows users to use these values for subscription filters.

* Added ability to duplicate live query results in Redis. When the `redis_duplicate_results` configuration option is set to `true`, all live query results will be copied to an additional Redis Pub/Sub channel named LQDuplicate.

* Added ability to controls the server-side HTTP keepalive property. Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

* Fixed an issue on the _Packs_ page in which Fleet would incorrectly handle the configured `server_url_prefix`.

## Fleet 3.11.0 (Apr 28, 2021)

* Improved Fleet performance by batch updating host seen time instead of updating synchronously. This improvement reduces MySQL CPU usage by ~33% with 4,000 simulated hosts and MySQL running in Docker.

* Added support for software inventory, introducing a list of installed software items on each host's respective _Host details_ page. This feature is flagged off by default (for now). Check out [the feature flag documentation for instructions on how to turn this feature on](https://fleetdm.com/docs/deploying/configuration#software-inventory).

* Added Windows support for `fleetctl` agent autoupdates. The `fleetctl updates` command provides the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Made running common queries more convenient by adding the ability to select a saved query directly from a host's respective _Host details_ page.

* Fixed an issue on the _Query_ page in which Fleet would override the CMD + L browser hotkey.

* Fixed an issue in which a host would display an unreasonable time in the "Last fetched" column.

## Fleet 3.10.1 (Apr 6, 2021)

* Fixed a frontend bug that prevented the "Pack" page and "Edit pack" page from rendering in the Fleet UI. This issue occurred when the `platform` key, in the requested pack's configuration, was set to any value other than `darwin`, `linux`, `windows`, or `all`.

## Fleet 3.10.0 (Mar 31, 2021)

* Added `fleetctl` agent auto-updates beta which introduces the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Added option for Identity Provider-Initiated (IdP-initiated) Single Sign-On (SSO).

* Improved logging. All errors are logged regardless of log level, some non-errors are logged regardless of log level (agent enrollments, runs of live queries etc.), and all other non-errors are logged on debug level.

* Improved login resilience by adding rate-limiting to login and password reset attempts and preventing user enumeration.

* Added Fleet version and Go version in the My Account page of the Fleet UI.

* Improved `fleetctl preview` to ensure the latest version of Fleet is fired up on every run. In addition, the Fleet UI is now accessible without having to click through browser security warning messages.

* Added prefer storing IPv4 addresses for host details.

## Fleet 3.9.0 (Mar 9, 2021)

* Added configurable host identifier to help with duplicate host enrollment scenarios. By default, Fleet's behavior does not change (it uses the identifier configured in osquery's `--host_identifier` flag), but for users with overlapping host UUIDs changing `--osquery_host_identifier` to `instance` may be helpful. 

* Made cool-down period for host enrollment configurable to control load on the database in scenarios in which hosts are using the same identifier. By default, the cooldown is off, reverting to the behavior of Fleet <=3.4.0. The cooldown can be enabled with `--osquery_enroll_cooldown`.

* Refreshed the Fleet UI with a new layout and horizontal navigation bar.

* Trimmed down the size of Fleet binaries.

* Improved handling of config_refresh values from osquery clients.

* Fixed an issue with IP addresses and host additional info dropping.

## Fleet 3.8.0 (Feb 25, 2021)

* Added search, sort, and column selection in the hosts dashboard.

* Added AWS Lambda logging plugin.

* Improved messaging about number of hosts responding to live query.

* Updated host listing API endpoints to support search.

* Added fixes to the `fleetctl preview` experience.

* Fixed `denylist` parameter in scheduled queries.

* Fixed an issue with errors table rendering on live query page.

* Deprecated `KOLIDE_` environment variable prefixes in favor of `FLEET_` prefixes. Deprecated prefixes continue to work and the Fleet server will log warnings if the deprecated variable names are used. 

* Deprecated `/api/v1/kolide` routes in favor of `/api/v1/fleet`. Deprecated routes continue to work and the Fleet server will log warnings if the deprecated routes are used. 

* Added Javascript source maps for development.

## Fleet 3.7.1 (Feb 3, 2021)

* Changed the default `--server_tls_compatibility` to `intermediate`. The new settings caused TLS connectivity issues for users in some environments. This new default is a more appropriate balance of security and compatibility, as recommended by Mozilla.

## Fleet 3.7.0 (Feb 3, 2021)

### This is a security release.

* **Security**: Fixed a vulnerability in which a malicious actor with a valid node key can send a badly formatted request that causes the Fleet server to exit, resulting in denial of service. See https://github.com/fleetdm/fleet/security/advisories/GHSA-xwh8-9p3f-3x45 and the linked content within that advisory.

* Added new Host details page which includes a rich view of a specific host’s attributes.

* Revealed live query errors in the Fleet UI and `fleetctl` to help target and diagnose hosts that fail.

* Added Helm chart to make it easier for users to deploy to Kubernetes.

* Added support for `denylist` parameter in scheduled queries.

* Added debug flag to `fleetctl` that enables logging of HTTP requests and responses to stderr.

* Improved the `fleetctl preview` experience to include adding containerized osquery agents, displaying login information, creating a default directory, and checking for Docker daemon status.

* Added improved error handling in host enrollment to make debugging issues with the enrollment process easier.

* Upgraded TLS compatibility settings to match Mozilla.

* Added comments in generated flagfile to add clarity to different features being configured.

* Fixed a bug in Fleet UI that allowed user to edit a scheduled query after it had been deleted from a pack.


## Fleet 3.6.0 (Jan 7, 2021)

* Added the option to set up an S3 bucket as the storage backend for file carving.

* Built Docker container with Fleet running as non-root user.

* Added support to read in the MySQL password and JWT key from a file.

* Improved the `fleetctl preview` experience by automatically completing the setup process and configuring fleetctl for users.

* Restructured the documentation into three top-level sections titled "Using Fleet," "Deployment," and "Contribution."

* Fixed a bug that allowed hosts to enroll with an empty enroll secret in new installations before setup was completed.

* Fixed a bug that made the query editor render strangely in Safari.

## Fleet 3.5.1 (Dec 14, 2020)

### This is a security release.

* **Security**: Introduced XML validation library to mitigate Go stdlib XML parsing vulnerability effecting SSO login. See https://github.com/fleetdm/fleet/security/advisories/GHSA-w3wf-cfx3-6gcx and the linked content within that advisory.

Follow up: Rotated `--auth_jwt_key` to invalidate existing sessions. Audit for suspicious activity in the Fleet server.

* **Security**: Prevents new queries from using the SQLite `ATTACH` command. This is a mitigation for the osquery vulnerability https://github.com/osquery/osquery/security/advisories/GHSA-4g56-2482-x7q8.

Follow up: Audit existing saved queries and logs of live query executions for possible malicious use of `ATTACH`. Upgrade osquery to 4.6.0 to prevent `ATTACH` queries from executing.

* Update icons and fix hosts dashboard for wide screen sizes.

## Fleet 3.5.0 (Dec 10, 2020)

* Refresh the Fleet UI with new colors, fonts, and Fleet logos.

* All releases going forward will have the fleectl.exe.zip on the release page.

* Added documentation for the authentication Fleet REST API endpoints.

* Added FAQ answers about the stress test results for Fleet, configuring labels, and resetting auth tokens.

* Fixed a performance issue users encountered when multiple hosts shared the same UUID by adding a one minute cooldown.

* Improved the `fleetctl preview` startup experience.

* Fixed a bug preventing the same query from being added to a scheduled pack more than once in the Fleet UI.


## Fleet 3.4.0 (Nov 18, 2020)

* Added NPM installer for `fleetctl`. Install via `npm install -g osquery-fleetctl`.

* Added `fleetctl preview` command to start a local test instance of the Fleet server with Docker.

* Added `fleetctl debug` commands and API endpoints for debugging server performance.

* Added additional_info_filters parameter to get hosts API endpoint for filtering returned additional_info.

* Updated package import paths from github.com/kolide/fleet to github.com/fleetdm/fleet.

* Added first of the Fleet REST API documentation.

* Added documentation on monitoring with Prometheus.

* Added documentation to FAQ for debugging database connection errors.

* Fixed fleetctl Windows compatibility issues.

* Fixed a bug preventing usernames from containing the @ symbol.

* Fixed a bug in 3.3.0 in which there was an unexpected database migration warning.

## Fleet 3.3.0 (Nov 05, 2020)

With this release, Fleet has moved to the new github.com/fleetdm/fleet
repository. Please follow changes and releases there.

* Added file carving functionality.

* Added `fleetctl user create` command.

* Added osquery options editor to admin pages in UI.

* Added `fleetctl query --pretty` option for pretty-printing query results. 

* Added ability to disable packs with `fleetctl apply`.

* Improved "Add New Host" dialog to walk the user step-by-step through host enrollment.

* Improved 500 error page by allowing display of the error.

* Added partial transition of branding away from "Kolide Fleet".

* Fixed an issue with case insensitive enroll secret and node key authentication.

* Fixed an issue with `fleetctl query --quiet` flag not actually suppressing output.


## Fleet 3.2.0 (Aug 08, 2020)

* Added `stdout` logging plugin.

* Added AWS `kinesis` logging plugin.

* Added compression option for `filesystem` logging plugin.

* Added support for Redis TLS connections.

* Added osquery host identifier to EnrollAgent logs.

* Added osquery version information to output of `fleetctl get hosts`.

* Added hostname to UI delete host confirmation modal.

* Updated osquery schema to 4.5.0.

* Updated osquery versions available in schedule query UI.

* Updated MySQL driver.

* Removed support for (previously deprecated) `old` TLS profile.

* Fixed cleanup of queries in bad state. This should resolve issues in which users experienced old live queries repeatedly returned to hosts. 

* Fixed output kind of `fleetctl get options`.

## Fleet 3.1.0 (Aug 06, 2020)

* Added configuration option to set Redis database (`--redis_database`).

* Added configuration option to set MySQL connection max lifetime (`--mysql_conn_max_lifetime`).

* Added support for printing a single enroll secret by name.

* Fixed bug with label_type in older fleetctl yaml syntax.

* Fixed bug with URL prefix and Edit Pack button. 

## Kolide Fleet 3.0.0 (Jul 23, 2020)

* Backend performance overhaul. The Fleet server can now handle hundreds of thousands of connected hosts.

* Pagination implemented in the web UI. This makes the UI usable for any host count supported by the backend.

* Added capability to collect "additional" information from hosts. Additional queries can be set to be updated along with the host detail queries. This additional information is returned by the API.

* Removed extraneous network interface information to optimize server performance. Users that require this information can use the additional queries functionality to retrieve it.

* Added "manual" labels implementation. Static labels can be set by providing a list of hostnames with `fleetctl`.

* Added JSON output for `fleetctl get` commands.

* Added `fleetctl get host` to retrieve details for a single host.

* Updated table schema for osquery 4.4.0.

* Added support for multiple enroll secrets.

* Logging verbosity reduced by default. Logs are now much less noisy.

* Fixed import of github.com/kolide/fleet Go packages for consumers outside of this repository.

## Kolide Fleet 2.6.0 (Mar 24, 2020)

* Added server logging for X-Forwarded-For header.

* Added `--osquery_detail_update_interval` to set interval of host detail updates.
  Set this (along with `--osquery_label_update_interval`) to a longer interval
  to reduce server load in large deployments.

* Fixed MySQL deadlock errors by adding retries and backoff to transactions.

## Kolide Fleet 2.5.0 (Jan 26, 2020)

* Added `fleetctl goquery` command to bring up the github.com/AbGuthrie/goquery CLI.

* Added ability to disable live queries in web UI and `fleetctl`.

* Added `--query-name` option to `fleetctl query`. This allows using the SQL from a saved query.

* Added `--mysql-protocol` flag to allow connection to MySQL by domain socket.

* Improved server logging. Add logging for creation of live queries. Add username information to logging for other endpoints.

* Allows CREATE queries in the web UI.

* Fixed a bug in which `fleetctl query` would exit before any results were returned when latency to the Fleet server was high.

* Fixed an error initializing the Fleet database when MySQL does not have event permissions.

* Deprecated "old" TLS profile.

## Kolide Fleet 2.4.0 (Nov 12, 2019)

* Added `--server_url_prefix` flag to configure a URL prefix to prepend on all Fleet URLs. This can be useful to run fleet behind a reverse-proxy on a hostname shared with other services.

* Added option to automatically expire hosts that have not checked in within a certain number of days. Configure this in the "Advanced Options" of "App Settings" in the browser UI.

* Added ability to search for hosts by UUID when targeting queries.

* Allows SAML IdP name to be as short as 4 characters.

## Kolide Fleet 2.3.0 (Aug 14, 2019)

### This is a security release.

* Security: Upgraded Go to 1.12.8 to fix CVE-2019-9512, CVE-2019-9514, and CVE-2019-14809.

* Added capability to export packs, labels, and queries as yaml in `fleetctl get` with the `--yaml` flag. Include queries with a pack using `--with-queries`.

* Modified email templates to load image assets from Github CDN rather than Fleet server (fixes broken images in emails when Fleet server is not accessible from email clients).

* Added warning in query UI when Redis is not functioning.

* Fixed minor bugs in frontend handling of scheduled queries.

* Minor styling changes to frontend.


## Kolide Fleet 2.2.0 (Jul 16, 2019)

* Added GCP PubSub logging plugin. Thanks to Michael Samuel for adding this capability.

* Improved escaping for target search in live query interface. It is now easier to target hosts with + and * characters in the name.

* Server and browser performance improved to reduced loading of hosts in frontend. Host status will only update on page load when over 100 hosts are present.

* Utilized details sent by osquery in enrollment request to more quickly display details of new hosts. Also fixes a bug in which hosts could not complete enrollment if certain platform-dependent options were used.

* Fixed a bug in which the default query runs after targets are edited.

## Kolide Fleet 2.1.2 (May 30, 2019)

* Prevented sending of SMTP credentials over insecure connection

* Added prefix generated SAML IDs with 'id' (improves compatibility with some IdPs)

## Kolide Fleet 2.1.1 (Apr 25, 2019)

* Automatically pulls AWS STS credentials for Firehose logging if they are not specified in config.

* Fixed bug in which log output did not include newlines separating characters.

* Fixed bug in which the default live query was run when navigating to a query by URL.

* Updated logic for setting primary NIC to ignore link-local or loopback interfaces.

* Disabled editing of logged in user email in admin panel (instead, use the "Account Settings" menu in top left).

* Fixed a panic resulting from an invalid config file path.

## Kolide Fleet 2.1.0 (Apr 9, 2019)

* Added capability to log osquery status and results to AWS Firehose. Note that this deprecated some existing logging configuration (`--osquery_status_log_file` and `--osquery_result_log_file`). Existing configurations will continue to work, but will be removed at some point.

* Automatically cleans up "incoming hosts" that do not complete enrollment.

* Fixed bug with SSO requests that caused issues with some IdPs.

* Hid built-in platform labels that have no hosts.

* Fixed references to Fleet documentation in emails.

* Minor improvements to UI in places where editing objects is disabled.

## Kolide Fleet 2.0.2 (Jan 17, 2019)

* Improved performance of `fleetctl query` with high host counts.

* Added `fleetctl get hosts` command to retrieve a list of enrolled hosts.

* Added support for Login SMTP authentication method (Used by Office365).

* Added `--timeout` flag to `fleetctl query`.

* Added query editor support for control-return shortcut to run query.

* Allowed preselection of hosts by UUID in query page URL parameters.

* Allowed username to be specified in `fleetctl setup`. Default behavior remains to use email as username.

* Fixed conversion of integers in `fleetctl convert`.

* Upgraded major Javascript dependencies.

* Fixed a bug in which query name had to be specified in pack yaml.

## Kolide Fleet 2.0.1 (Nov 26, 2018)

* Fixed a bug in which deleted queries appeared in pack specs returned by fleetctl.

* Fixed a bug getting entities with spaces in the name.

## Kolide Fleet 2.0.0 (Oct 16, 2018)

* Stable release of Fleet 2.0.

* Supports custom certificate authorities in fleetctl client.

* Added support for MySQL 8 authentication methods.

* Allows INSERT queries in editor.

* Updated UI styles.

* Fixed a bug causing migration errors in certain environments.

See changelogs for release candidates below to get full differences from 1.0.9
to 2.0.0.

## Kolide Fleet 2.0.0 RC5 (Sep 18, 2018)

* Fixed a security vulnerability that would allow a non-admin user to elevate privileges to admin level.

* Fixed a security vulnerability that would allow a non-admin user to modify other user's details.

* Reduced the information that could be gained by an admin user trying to port scan the network through the SMTP configuration.

* Refactored and add testing to authorization code.

## Kolide Fleet 2.0.0 RC4 (August 14, 2018)

* Exposed the API token (to be used with fleetctl) in the UI.

* Updated autocompletion values in the query editor.

* Fixed a longstanding bug that caused pack targets to sometimes update incorrectly in the UI.

* Fixed a bug that prevented deletion of labels in the UI.

* Fixed error some users encountered when migrating packs (due to deleted scheduled queries).

* Updated favicon and UI styles.

* Handled newlines in pack JSON with `fleetctl convert`.

* Improved UX of fleetctl tool.

* Fixed a bug in which the UI displayed the incorrect logging type for scheduled queries.

* Added support for SAML providers with whitespace in the X509 certificate.

* Fixed targeting of packs to individual hosts in the UI.

## Kolide Fleet 2.0.0 RC3 (June 21, 2018)

* Fixed a bug where duplicate queries were being created in the same pack but only one was ever delivered to osquery. A migration was added to delete duplicate queries in packs created by the UI.
  * It is possible to schedule the same query with different options in one pack, but only via the CLI.
  * If you thought you were relying on this functionality via the UI, note that duplicate queries will be deleted when you run migrations as apart of a cleanup fix. Please check your configurations and make sure to create any double-scheduled queries via the CLI moving forward.

* Fixed a bug in which packs created in UI could not be loaded by fleetctl.

* Fixed a bug where deleting a query would not delete it from the packs that the query was scheduled in.

## Kolide Fleet 2.0.0 RC2 (June 18, 2018)

* Fixed errors when creating and modifying packs, queries and labels in UI.

* Fixed an issue with the schema of returned config JSON.

* Handled newlines when converting query packs with fleetctl convert.

* Added last seen time hover tooltip in Fleet UI.

* Fixed a null pointer error when live querying via fleetctl.

* Explicitly set timezone in MySQL connection (improves timestamp consistency).

* Allowed native password auth for MySQL (improves compatibility with Amazon RDS).

## Kolide Fleet 2.0.0 (currently preparing for release)

The primary new addition in Fleet 2 is the new `fleetctl` CLI and file-format, which dramatically increases the flexibility and control that administrators have over their osquery deployment. The CLI and the file format are documented [in the Fleet documentation](https://fleetdm.com/docs/using-fleet/fleetctl-cli).

### New Features

* New `fleetctl` CLI for managing your entire osquery workflow via CLI, API, and source controlled files!
  * You can use `fleetctl` to manage osquery packs, queries, labels, and configuration.

* In addition to the CLI, Fleet 2.0.0 introduces a new file format for articulating labels, queries, packs, options, etc. This format is designed for composability, enabling more effective sharing and re-use of intelligence.

```yaml
apiVersion: v1
kind: query
spec:
  name: pending_updates
  query: >
    select value
    from plist
    where
      path = "/Library/Preferences/ManagedInstalls.plist" and
      key = "PendingUpdateCount" and
      value > "0";
```

* Run live osquery queries against arbitrary subsets of your infrastructure via the `fleetctl query` command.

* Use `fleetctl setup`, `fleetctl login`, and `fleetctl logout` to manage the authentication life-cycle via the CLI.

* Use `fleetctl get`, `fleetctl apply`, and `fleetctl delete` to manage the state of your Fleet data.

* Manage any osquery option you want and set platform-specific overrides with the `fleetctl` CLI and file format.

### Upgrade Plan

* Managing osquery options via the UI has been removed in favor of the more flexible solution provided by the CLI. If you have customized your osquery options with Fleet, there is [a database migration](./server/datastore/mysql/migrations/data/20171212182458_MigrateOsqueryOptions.go) which will port your existing data into the new format when you run `fleet prepare db`. To download your osquery options after migrating your database, run `fleetctl get options > options.yaml`. Further modifications to your options should occur in this file and it should be applied with `fleetctl apply -f ./options.yaml`.

## Kolide Fleet 1.0.8 (May 3, 2018)

* Osquery 3.0+ compatibility!

* Included RFC822 From header in emails (for email authentication)

## Kolide Fleet 1.0.7 (Mar 30, 2018)

* Now supports FileAccesses in FIM configuration.

* Now populates network interfaces on windows hosts in host view.

* Added flags for configuring MySQL connection pooling limits.

* Fixed bug in which shard and removed keys are dropped in query packs returned to osquery clients.

* Fixed handling of status logs with unexpected fields.

## Kolide Fleet 1.0.6 (Dec 4, 2017)

* Added remote IP in the logs for all osqueryd/launcher requests. (#1653)

* Fixed bugs that caused logs to sometimes be omitted from the logwriter. (#1636, #1617)

* Fixed a bug where request bodies were not being explicitly closed. (#1613)

* Fixed a bug where SAML client would create too many HTTP connections. (#1587)

* Fixed bug in which default query was run instead of entered query. (#1611)

* Added pagination to the Host browser pages for increased performance. (#1594)

* Fixed bug rendering hosts when clock speed cannot be parsed. (#1604)

## Kolide Fleet 1.0.5 (Oct 17, 2017)

* Renamed the binary from kolide to fleet.

* Added support for Kolide Launcher managed osquery nodes.

* Removed license requirements.

* Updated documentation link in the sidebar to point to public GitHub documentation.

* Added FIM support.

* Title on query page correctly reflects new or edit mode.

* Fixed issue on new query page where last query would be submitted instead of current.

* Fixed issue where user menu did not work on Firefox browser.

* Fixed issue cause SSO to fail for ADFS.

* Fixed issue validating signatures in nested SAML assertions..

## Kolide 1.0.4 (Jun 1, 2017)

* Added feature that allows users to import existing Osquery configuration files using the [configimporter](https://github.com/kolide/configimporter) utility.

* Added support for Osquery decorators.

* Added SAML single sign on support.

* Improved online status detection.

  The Kolide server now tracks the `distributed_interval` and `config_tls_refresh` values for each individual host (these can be different if they are set via flagfile and not through Kolide), to ensure that online status is represented as accurately as possible.

* Kolide server now requires `--auth_jwt_key` to be specified at startup.

  If no JWT key is provided by the user, the server will print a new suggested random JWT key for use.

* Fixed bug in which deleted packs were still displayed on the query sidebar.

* Fixed rounding error when showing % of online hosts.

* Removed --app_token_key flag.

* Fixed issue where heavily loaded database caused host authentication failures.

* Fixed issue where osquery sends empty strings for integer values in log results.

## Kolide 1.0.3 (April 3, 2017)

* Log rotation is no longer the default setting for Osquery status and results logs. To enable log rotation use the `--osquery_enable_log_rotation` flag.

* Added a debug endpoint for collecting performance statistics and profiles.

  When `kolide serve --debug` is used, additional handlers will be started to provide access to profiling tools. These endpoints are authenticated with a randomly generated token that is printed to the Kolide logs at startup. These profiling tools are not intended for general use, but they may be useful when providing performance-related bug reports to the Kolide developers.

* Added a workaround for CentOS6 detection.

  Osquery 2.3.2 incorrectly reports an empty value for `platform` on CentOS6 hosts. We added a workaround to properly detect platform in Kolide, and also [submitted a fix](https://github.com/facebook/osquery/pull/3071) to upstream osquery.

* Ensured hosts enroll in labels immediately even when `distributed_interval` is set to a long interval.

* Optimizations reduce the CPU and DB usage of the manage hosts page.

* Managed packs page now loads much quicker when a large number of hosts are enrolled.

* Fixed bug with the "Reset Options" button.

* Fixed 500 error resulting from saving unchanged options.

* Improved validation for SMTP settings.

* Added command line support for `modern`, `intermediate`, and `old` TLS configuration
profiles. The profile is set using the following command line argument.
```
--server_tls_compatibility=modern
```
See https://wiki.mozilla.org/Security/Server_Side_TLS for more information on the different profile options.

* The Options Configuration item in the sidebar is now only available to admin users.

  Previously this item was visible to non-admin users and if selected, a blank options page would be displayed since server side authorization constraints prevent regular users from viewing or changing options.

* Improved validation for the Kolide server URL supplied in setup and configuration.

* Fixed an issue importing osquery configurations with numeric values represented as strings in JSON.

## Kolide 1.0.2 (March 14, 2017)

* Fixed an issue adding additional targets when querying a host

* Shows loading spinner while newly added Host Details are saved

* Shows a generic computer icon when when referring to hosts with an unknown platform instead of the text "All"

* Kolide will now warn on startup if there are database migrations not yet completed.

* Kolide will prompt for confirmation before running database migrations.

  To disable this, use `kolide prepare db --no-prompt`.

* Kolide now supports emoji, so you can 🔥 to your heart's content.

* When setting the platform for a scheduled query, selecting "All" now clears individually selected platforms.

* Updated Host details cards UI

* Lowered HTTP timeout settings.

  In an effort to provide a more resilient web server, timeouts are more strictly enforced by the Kolide HTTP server (regardless of whether or not you're using the built-in TLS termination).

* Hardened TLS server settings.

  For customers using Kolide's built-in TLS server (if the `server.tls` configuration is `true`), the server was hardened to only accept modern cipher suites as recommended by [Mozilla](https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility).

* Improve the mechanism used to calculate whether or not hosts are online.

  Previously, hosts were categorized as "online" if they had been seen within the past 30 minutes. To make the "online" status more representative of reality, hosts are marked "online" if the Kolide server has heard from them within two times the lowest polling interval as described by the Kolide-managed osquery configuration. For example, if you've configured osqueryd to check-in with Kolide every 10 seconds, only hosts that Kolide has heard from within the last 20 seconds will be marked "online".

* Updated Host details cards UI

* Added support for rotating the osquery status and result log files by sending a SIGHUP signal to the kolide process.

* Fixed Distributed Query compatibility with load balancers and Safari.

  Customers running Kolide behind a web balancer lacking support for websockets were unable to use the distributed query feature. Also, in certain circumstances, Safari users with a self-signed cert for Kolide would receive an error. This release add a fallback mechanism from websockets using SockJS for improved compatibility.

* Fixed issue with Distributed Query Pack results full screen feature that broke the browser scrolling abilities.

* Fixed bug in which host counts in the sidebar did not match up with displayed hosts.

## Kolide 1.0.1 (February 27, 2017)

* Fixed an issue that prevented users from replacing deleted labels with a new label of the same name.

* Improved the reliability of IP and MAC address data in the host cards and table.

* Added full screen support for distributed query results.

* Enabled users to double click on queries and packs in a table to see their details.

* Reprompted for a password when a user attempts to change their email address.

* Automatically decorates the status and result logs with the host's UUID and hostname.

* Fixed an issue where Kolide users on Safari were unable to delete queries or packs.

* Improved platform detection accuracy.

  Previously Kolide was determining platform based on the OS of the system osquery was built on instead of the OS it was running on. Please note: Offline hosts may continue to report an erroneous platform until they check-in with Kolide.

* Fixed bugs where query links in the pack sidebar pointed to the wrong queries.

* Improved MySQL compatibility with stricter configurations.

* Allows users to edit the name and description of host labels.

* Added basic table autocompletion when typing in the query composer.

* Now support MySQL client certificate authentication. More details can be found in the [Configuring the Fleet binary docs](./docs/infrastructure/configuring-the-fleet-binary.md).

* Improved security for user-initiated email address changes.

  This improvement ensures that only users who own an email address and are logged in as the user who initiated the change can confirm the new email.

  Previously it was possible for Administrators to also confirm these changes by clicking the confirmation link.

* Fixed an issue where the setup form rejects passwords with certain characters.

  This change resolves an issue where certain special characters like "." where rejected by the client-side JS that controls the setup form.

* Now automatically logs in the user once initial setup is completed.
