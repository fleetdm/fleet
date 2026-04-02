# Fleet 4.82.0 | Fleets and reports, new technician role, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/zeU1IdlyxIY?si=bjoX3-kh8wVN7ECh" title="0" allowfullscreen></iframe>
</div>

Fleet 4.82.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.82.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Fleets and reports rename
- New technician role for helpdesk teams
- Self-service scripts on macOS
- Fully-managed Android

### Fleets and reports rename

Fleet now uses "fleets" instead of "teams" and "reports" instead of "queries" across the UI, API, CLI, and GitOps (YAML). The new "fleets" terminology better reflects how hosts are grouped and managed in Fleet. "Reports" makes it clearer that these are used to collect host information.

Existing workflows continue to work. All API endpoints, CLI commands, and YAML Keys with "teams" and "queries" are still supported for backward compatibility and automatically map to "fleets" and "reports" respectively. Reference documentation updates are coming soon.

Deprecation warning are coming to fleetctl and Fleet server logs in the next release (4.83). If you want to turn them on early, see the [fleectl](https://fleetdm.com/guides/fleetctl#deprecation-warnings) and [server logs](http://fleetdm.com/docs/configuration/fleet-server-configuration#logging-enable-topics) instructions.

GitHub issues: [#39314](https://github.com/fleetdm/fleet/issues/39314), [#39238](https://github.com/fleetdm/fleet/issues/39238) 

### New technician role

Fleet now includes a Technician role designed for helpdesk and IT support teams. Technicians can run scripts, view results, and install or uninstall software. Check out the [permissions table](https://fleetdm.com/guides/role-based-access#user-permissions) for a full list of permissions.

This enables least-privilege access for day-to-day support tasks while keeping sensitive configuration settings restricted.

GitHub issue: [#35696](https://github.com/fleetdm/fleet/issues/35696)

### Self-service scripts on macOS

Fleet now supports self-service scripts through script-only packages on macOS. Upload a `.sh` script and make it available in self-service for end users on macOS hosts. [Learn how](https://fleetdm.com/guides/software-self-service).

This makes it easier for IT admins to deliver quick fixes and utility scripts.

GitHub issue: [#33951](https://github.com/fleetdm/fleet/issues/33951)

### Fully-managed Android

Fleet now supports managing company-owned Android hosts in fully-managed mode. This allows IT teams to apply stricter controls and use Android management features that aren’t available on BYOD Android hosts (work profiles). Learn how to [enroll Android hosts](https://fleetdm.com/guides/enroll-hosts).

GitHub issue: [#36337](https://github.com/fleetdm/fleet/issues/36337)

## Changes

### IT Admins
- Added support for enrolling fully managed Android hosts without a work profile.
- Added capability to uninstall Android apps on the device (and removal from self-service in the managed Google Play store) when an app is removed from Fleet.
- Added ability to allow or disallow end-users to bypass conditional access on a per-policy basis.
- Added filtering by platform and add status to the Software > Add Fleet-maintained apps table.
- Updated Android status reports to re-verify profiles that previously failed.
- Added ability to roll back to previously added versions of Fleet-maintained apps.
- Added new Technician role designed for help desk and IT support teams. Technicians can run scripts, view results, and install or uninstall software.
- Added support for JIT provisioning of the Technician role via SSO SAML attributes.
- Added automatic retries for failed software operations.

### Security Engineers
- Added ability to scan for kernel vulnerabilities on RHEL based hosts.
- Added AWS GovCloud RDS CA certificates to the RDS MySQL TLS bundle, enabling IAM authentication for Fleet deployments connecting to RDS in AWS GovCloud regions (us-gov-east-1, us-gov-west-1).
- Added CVE alias for python visual studio code extension.
- Added new activity for edited enroll secrets.

### Other improvements and bug fixes
- Renamed teams and queries to fleets and reports in the UI, API, CLI, and GitOps.
- Deprecated no-team.yml in GitOps in favor of unassigned.yml.
- Deprecated certain API field names to reflect the renaming of "teams" to "fleets" and "queries" to "reports".
- Updated Android MDM profiles to show up as pending on upload, the same as Apple MDM profiles.
- Improved the speed of a database query that runs every minute to avoid database locking.
- Added configurable body size limits for the `/api/osquery/log` and `/api/osquery/distributed/write` endpoints.
- Updated logic to trigger vulnerability webhook when on Fleet free tier.
- Updated storage of the auth token used in the UI.
- Dynamically alphabetized vitals on the host details page.
- Reworked how we handle server/worker delays to fix flaky tests.
- Disabled "Calendar" dropdown option in Policy > Manage automations for Unassigned.
- Added Go slog logging infrastructure and migrated a portion of the code from go-kit/log to slog.
- Added CTA to turn on Android MDM for Android software setup experience if MDM is not configured.
- Left-aligned "Critical" checkbox in Save policy form.
- Improved spacing on the Controls > OS Settings page.
- Updated to not allow editing Fleet-maintained app in the UI while GitOps mode is enabled.
- Updated to accept the previous device authentication token for up to one rotation cycle, so the My Device page URL remains valid after token refresh.
- Updated default macOS, iOS, and iPadOS update deadline time to 7PM (19:00) local time.
- Updated UI to enable adding/removing multiple Microsoft Entra tenant ids.
- Added additional logging for SCEP proxy requests and SCEP profile renewals.
- Added warning message on gitops label rename to clarify to users that renaming a label implies a delete operation.
- Added the ability to specify allowed Entra tenant IDs for enrollments.
- Updated the DEP syncer to properly reassign a profile when ABM unilaterally removes it.
- Increased the maximum script execution timeout from 1 hour (3600 seconds) to 5 hours (18000 seconds).
- Improved error handling on AWS DB failover. Fleet will now fail health check if the primary DB is read-only, or trigger graceful shutdown when write operations encounter read-only errors.
- Generated a server-side device token in the Okta conditional access flow when none exists or the current token is expired.
- Moved the copy button for text areas out of the text area itself and in line with its label.
- Removed unnecessary calls to `svc.ds.BulkSetPendingMDMHostProfiles` in `POST /api/latest/fleet/spec/fleets`.
- Internal refactoring: moved `/api/_version_/fleet/hosts/{id:[0-9]+}/activities` endpoint and `MarkActivitiesAsStreamed` to new server/activity bounded context.
- Added `logging.otel_logs_enabled` contributor config option to export server logs to OpenTelemetry.
- Added automatic tagging of prerelease/post-release versions on local build based on branch name.
- Added ability to enable/disable logs by topic.
- Improved detection of `DISPLAY` variable in X11 sessions.
- Updated the "Used by" column heading on the hosts page to "User email".
- Refactored query used for deleting host_mdm_apple_profiles in bulk to use Primary keys only.
- Added `team_id` to host details page param in URL to allow retaining team on refresh.
- Added help text on the software details page, below the installer status table, to explain the meanings of the counts.
- Added Country:US to new CA certs created by Fleet.
- Added error if GitOps/batch attempts to add setup experience software when manual agent install is enabled.
- Updated "Manage automations" button on the Queries and Policies pages to now always be visible, and disabled only when the current team has no queries of its own.
- Updated validation rules around the creation of labels to make sure only valid platforms are used.
- Improved host software inventory table's handling of long "Type" values.
- Updated expiration date of the auth token cookie to match the fleet session duration.
- Surfaced FMA version used and whether it's out of date in the UI.
- Updated nats-server dependency to resolve dependency vulnerabilities.
- Improved validation for host transfers.
- Fixed matching logic on App component for pages titles.
- Fixed adding Windows Fleet maintained apps failing when a software title with the same upgrade code already exists.
- Fixed an issue where GitOps would not respect the value set on `update_new_hosts` for macOS updates.
- Fixed an issue where duplicate kernels were reported in the OS versions API for RHEL-family distributions (RHEL, AlmaLinux, CentOS, Rocky, Fedora).
- Fixed issue where Windows Jetbrains products would not report the correct version number.
- Fixed a bug where custom software installer display names and icons were not used in the setup experience UI.
- Fixed a bug where the list activities API endpoint would fail with a database error when there were more than 65,535 activities and no pagination parameters were specified. The maximum `per_page` for activities endpoints is now 10,000.
- Fixed issue where MySQL IAM authentication could fail when a custom TLS CA/TLS config was set (for example GovCloud), by ensuring Fleet includes the configured TLS mode in IAM DSNs.
- Fixed styling issues for the UI when no enroll secret is present on a fleet.
- Fixed an issue where some UI users saw a blank gutter on the right side of parts of the UI.
- Fixed a bug where certain macOS app names could be ingested as empty strings due to incorrect ".app" suffix removal.
- Fixed install/uninstall tarballs package to skip recently updated status that is waiting for a change in software inventory
- Fixed a bug where software installers could create titles with the wrong platform.
- Fixed a bug where 2 vulnerability jobs can run in parallel if one is taking longer than 2 hours.
- Fixed issue with hosts incorrectly reporting policy failures after policy label targets changed.
- Fixed client-side errors being incorrectly reported as server errors in OTEL telemetry.
- Fixed issue where the status name was wrapping at smaller viewport widths on the mdm card on the Dashboard page.
- Fixed false negative CVE-2026-20841 on Windows Notepad.
- Fixed false positive CVE for Nextcloud Desktop.
- Fixed rare CPE error when software name sanitizes to empty (e.g. only special characters).
- Fixed Android enrollment to associate hosts with SCIM users, populating full name, groups, and department in host vitals.
- Fixed a hover style issue in the label filter close button.
- Fixed mismatches between disk encryption summary counts vs hosts displayed.
- Fixed truncation of certificate fields containing non-ASCII characters.
- Fixed an issue where policy automation settings in the Other Workflows modal reverted to stale values after saving when using a MySQL read replica.
- Fixed query results cleanup cron failing with "too many placeholders" error by filtering to only saved queries and batching the SQL IN clause.
- Fixed DB lock contention during vulnerability cron's software cleanup that caused failures under load.
- Fixed pagination on the host software page incorrectly disabling the "Next" button when a software title has multiple installer versions.
- Fixed a bug where macOS systems previous enrolled in fleet wouldn't always go through setup experience after a wipe
- Fixed stale software titles list after adding a VPP or fleet-maintained app by invalidating the query cache on success.
- Fixed issue where Windows Jetbrains products would not report the correct version number.
- Fixed false positive `PayloadTooLargeError` errors.
- Fixed software appearance edits not reflected until page refresh.
- Fixed issue where policy automation retries were potentially reading stale data from replica database.
- Fixed label edits not reflected until page refresh.
- Fixed report creation API returning zero timestamps for `created_at` and `updated_at` fields.
- Fixed issue where arbitrary order_key values could be used to extract data.
- Fixed stale software titles list after deleting a software installer.
- Fixed query results cleanup cron failing with "too many placeholders" error by filtering to only saved queries and batching the SQL IN clause.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.81.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-03-11">
<meta name="articleTitle" value="Fleet 4.82.0 | Fleets and reports, new technician role, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.82.0-1600x900@2x.png">
