# Fleet 4.65.0 | GitOps mode, automatically install software, certificates in host vitals

Fleet 4.65.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.65.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- GitOps mode
- Automatically install software
- Certificates in host vitals

### GitOps mode

You can now put Fleet in "GitOps mode" which puts the Fleet UI in a read-only mode that prevents edits. This points users in the UI to your git repo and ensures that changes aren’t accidentally overwritten by your GitHub action or GitLab CI/CD gitops runs.

### Automatically install software

Fleet now allows IT admins to [install App Store apps automatically](https://fleetdm.com/guides/automatic-software-install-in-fleet) on all your hosts without writing custom policies. This saves time when deploying apps across many hosts, making large-scale app deployment easier and more reliable.

### Certificates in host vitals

The **Host details** page now includes [certificates](https://fleetdm.com/vitals/host-certificates-mac-os#apple) for macOS, iOS, and iPadOS hosts as part of [host vitals](https://fleetdm.com/vitals). This helps IT teams quickly diagnose Wi-Fi or VPN connection issues by identifying missing or expired certificates that may be preventing network access.

## Changes

### Security
- Added UI for viewing certificate details on the host details and my device pages.
- Added new features to include certificates in host vitals for macOS, iOS, and iPadOS.
- Added the list host certificates (and list device's certificates) endpoints.
- Improved the copy for the delete and transfer host modal to be more clear about the disk encryption key behavior.
- Permit setting SSO metadata and metadata_url in gitops and UI.
- Fixed an issue where the Show Query modal would truncate large queries.
- Fixed Python for Windows software version mutation to avoid panics on software ingestion in some cases.
- Prevented an invalid `FLEET_VULNERABILITIES_MAX_CONCURRENCY` value from causing deadlocks during vulnerability processing.
- Updated default for vulnerabilities max concurrency from 5 to 1.
- Updated CPE generation to more closely align with CPEs use in vulnerability feeds.
- Changed software version CVE resolved in version parsing and comparison to use custom code rather than semver.
- Added new (as of 2025-03-07) archives page to data source for MS Mac Office vulnerability feed (applies to vulnerabilities feed rather than a specific Fleet release).
- Fixed an issue with Fleet's processing of Python versions to ensure that the correct CPEs are checked for vulnerabilities.
- Fixed an issue with increased resource usage during vulnerabilities processing by adding database indexes.
- Fixed false-positives on released PowerShell versions for CVE-2025-21171 and all PowerShell versions on CVE-2023-48795.

### IT
- Implemented GitOps mode that locks settings in the UI that are managed by GitOps.
- Allowed VPP apps to be automatically installed via a Fleet-created policy. 
- Added ability for users to automatically install App Store Apps without writing a policy in the Fleet UI.
- Updated the UI for adding and editing software for a cleaner, cohesive experience.
- Added auto-install to FMA via the API, replacing a more brittle client-side implementation.
- Added pagination inside each of the Manage Automations modals for policies.
- Added script execution to the new `upcoming_activities` table.
- Added software installs to the new `upcoming_activities` table.
- Added vpp apps installs to the new `upcoming_activities` table.
- Updated the list upcoming activities endpoint to use the new `upcoming_activities` table as source of truth.
- Added support to activate the next activity when one is enqueued or when one is completed.
- Added UI to the BYOD enrollment page to support enrolling Android devices into Fleet MDM.
- Added UI to turn on and off Android MDM.
- Added Android MDM activities.
> **NOTE:** Android features are currently experimental and disabled by default. To enable, set `ANDROID_FEATURE_ENABLED=1`.
- Updated UI for device user page with improved instructions for turning on MDM.
- Added `PATCH /api/latest/fleet/software/titles/:id/name` endpoint for cleaning up incorrect software titles for software that has a bundle ID.
- Added a daily job that keeps the App Store app version displayed in Fleet in sync with the actual latest version.
- Properly re-routed deleting a app on no team to no team software page insteal of all teams software page.
- Added a DB migration to migrate existing pending activities to the new unified queue.
- Added created_at timestamp for when a VPP app was added to a specific team.
> **NOTE:** The database migration for the above hydrates timestamps for existing VPP app team associations based on when the associated VPP apps were first added to the database. To hydrate more accurate timestamps by pulling from VPP app add/edit activities, you can run the following query manually. It is not included in migrations as it requires full table scans of the `activities` table, which may result in long migration times.
```sql
UPDATE vpp_apps_teams vat
LEFT JOIN (SELECT MAX(created_at) added_at, details->>"$.app_store_id" adam_id, details->>"$.platform" platform, details->>"$.team_id" team_id
    	FROM activities WHERE activity_type = 'added_app_store_app' GROUP BY adam_id, platform, team_id) aa ON
	vat.global_or_team_id = aa.team_id AND vat.adam_id = aa.adam_id AND vat.platform = aa.platform
LEFT JOIN (SELECT MAX(created_at) edited_at, details->>"$.app_store_id" adam_id, details->>"$.platform" platform, details->>"$.team_id" team_id
		FROM activities WHERE activity_type = 'edited_app_store_app' GROUP BY adam_id, platform, team_id) ae ON
	vat.global_or_team_id = ae.team_id AND vat.adam_id = ae.adam_id AND vat.platform = ae.platform
SET vat.created_at = COALESCE(added_at, vat.created_at), vat.updated_at = COALESCE(edited_at, added_at, vat.updated_at);
```
- Fixed an issue with assigning Windows MDM profiles to large numbers (> 65k) of hosts by batching the relevant database queries.
- Fixed policy software automation that falsely reported success in UI when updates actually failed. Users will now be properly notified of failed automation saves.
- Fixed a bug where uploading a macOS installer could prevent the software from being inventoried.
- Fixed a bug where target selector was present in a premature stage.
- Fixed a bug that caused macOS App Store apps to show up in Fleet as Windows apps if the Windows ersion of the app was already in Fleet.
- Fixed an issue where the ABM token teams were being reset when making updates to the app config.
- Fixed parsing of relative paths for MDM profiles in gitops `no-team.yml`.
- Fixed a bug where new `fleetd` could not install software from old fleet server.
- Fixed issue where `fleetctl gitops` was NOT deleting macOS setup experience bootstrap package and enrollment profile. GitOps should clear all settings that are not explicitly set in YAML config files.

### Bug fixes and improvements
- Set collation and character set explicitly on database tables that were missing explicit values.
- Updated the copy printed on successful runs of `fleetctl package`.
- Enabled redis cluster follow redierctions by default.
- Switched to a simpler, more reliable query for checking if an initial admin user has been added.
- Updated the styling of the "Used by" line on host details page to be easier to read and include more data in the tooltip.
- Added constistent behavior for table overflow and not hiding badges when user names overflow table cell.
- Updated wine to version 10.0 to improve support macOS-to-Windows installer creation on M1 chips.
- Updated UI to always show "Manage Automations" to permitted users.
- Fixed clicking "Show details" to open the software details modal on the My device page. 
- Fixed an issue where link protection services would prematurely redeem MFA links.
- Fixed several links that were dropping team_id parameters resetting team to all teams.
- Fixed password authentication getting disabled when SMTP isn't configured.
- Fixed an issue where restarting the desktop manager on Ubuntu would cause the Fleet Desktop tray icon to disappear and not return.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.65.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-03-14">
<meta name="articleTitle" value="Fleet 4.65.0 | GitOps mode, automatically install software, certificates in host vitals">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.65.0-1600x900@2x.png">
