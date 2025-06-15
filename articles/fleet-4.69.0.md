# Fleet 4.69.0 | Bulk scripts improvements, Entra ID and authentik foreign vitals, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/KfWGkgaMEN0?si=XpL8tufModTR9Q_O" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.69.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.69.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

If you're using GitOps to manage Fleet, read the note on [global configuration in GitOps](#global-configuration-in-gitops) before upgrading.

## Highlights

- Bulk scripts improvements
- Entra ID and authentik foreign vitals
- Secondary CVSS scores
- Self-service software: uninstall
- Add custom packages in GitOps mode
- Bulk resend failed configuration profiles
- Turn off MDM on iOS/iPadOS

### Bulk scripts improvements

IT Admins can now [run scripts in bulk](https://fleetdm.com/guides/scripts#batch-execute-scripts) using host filters. This makes it easy to target and take action on hundreds or more hosts without manually selecting them.

### Entra ID and authentik foreign vitals

Fleet now supports [pulling user data](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts)—like IdP email, full name, and groups—from [Entra ID](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts#microsoft-entra-id) or [authentik](https://goauthentik.io/) into host vitals. This helps IT Admins quickly identify the user assigned to each host.

### Secondary CVSS scores

When a vulnerability has no primary CVSS score in the [National Vulnerability Database (NVD)](https://nvd.nist.gov/), Fleet now shows the secondary score instead. This gives Security Engineers better visibility into potential risk and helps prioritize remediation.

### Add custom packages in GitOps mode

In GitOps mode, IT Admins can now use the UI to add a custom package and copy the corresponding YAML. This is useful for managing private software (like CrowdStrike) without a public URL.

### Bulk resend failed configuration profiles

IT Admins can now see all hosts that failed to apply a configuration profile and resend it in one step. No need to visit each host’s **Host details** page to retry.

### Turn off MDM on iOS/iPadOS

IT Admins can now disable MDM directly from the host detail page. This makes managing MDM status more consistent across all Apple devices in your fleet.

## Changes

### Security Engineers
- Added vulnerability detection via OVAL for Ubuntu 24.10 and 25.04.
- Added ability to sync end user's IdP information with Microsoft Entra ID using SCIM protocol.
- Added ability to sync end user's IdP information with Authentik using SCIM protocol.
- Updated Windows 11 Enterprise CIS policies to version 4.0.
- Added new Detail Query 'luks_verify' used to verify if the stored LUKS key is valid.
- Added additional checks to vulnerability feed validation to prevent deploying an un-enriched NVD feed.
- Added SHA256 hash of Mac applications to signature information in host software response.
- Added `FLEET_AUTH_SSO_SESSION_VALIDITY_PERIOD` environment variable for overriding how long end users have to complete SSO.
- Added ability to execute scripts on up to 5,000 hosts at a time using filters.
- Added ability to run a script on all hosts that match the current set of supported filters.
- Added a new API `GET /scripts/batch/summary/:batch_execution_id` endpoint for retrieving a summary of the current state of a batch script execution.
- Added the endpoint `POST /api/v1/fleet/configuration_profiles/resend/batch` to resend a profile to all hosts that satisfy the filter.
- Added a starter library that is automatically applied to all new Fleet instances during setup.

### IT Admins
- Added ability to execute scripts on up to 5,000 hosts at a time using filters.
- Added ability to run a script on all hosts that match the current set of supported filters.
- Added a new API `GET /scripts/batch/summary/:batch_execution_id` endpoint for retrieving a summary of the current state of a batch script execution.
- Added the endpoint `POST /api/v1/fleet/configuration_profiles/resend/batch` to resend a profile to all hosts that satisfy the filter.
- Added ability to uninstall software via Self-service tab of My device.
- Added a starter library that is automatically applied to all new Fleet instances during setup.
- Added `FLEET_MDM_SSO_RATE_LIMIT_PER_MINUTE` environment variable to allow increasing MDM SSO endpoint rate limit from 10 per minute. When supplied, this parameter also splits MDM SSO into its own rate limit bucket (default is shared with login endpoints).
- Added ability to sync end user's IdP information with Microsoft Entra ID using SCIM protocol.
- Added ability to sync end user's IdP information with Authentik using SCIM protocol.
- Updated Apple MDM enrollment to skip webview popup when end user authentication is disabled.
- Added SHA256 hash of Mac applications to signature information in host software response.
- Added UI to filter hosts by config profile status.
- Added UI for seeing custom profile status and to batch resend to hosts its failed on.
- Added filtering for hosts endpoints by MFM config profile and status.
- Added immediate cancellation of profile delivery when a profile is deleted; if it had already been installed then its removal will be pending.
- Added ability to turn off MDM for iPhone and iPad hosts on the hosts details page.
- Added ability for gitops mode to add a custom package on the software page to then copy/paste the YAML needed for packages that cannot be referenced with a URL.

### Global configuration in GitOps

This release fixed issue where SSO settings, SMTP settings, Features and MDM end-user authentication settings would not be cleared if they were omitted from YAML files used in a GitOps run. 

**If you have these settings configured via the Fleet web application and you use GitOps to manage your configuration, be sure settings are present in your global YAML settings file before your next GitOps run.**

### Other improvements and bug fixes
- Added Neon to the list of platforms that are detected as Linux distributions.
- Updated scripts so that editing will now cancel queued executions.
- Warn users of consequences when updating script contents.
- Improved effectiveness of app-wide text-truncation-into-tooltip functionality.
- Prevented misleading UI when a saved script's contents have changed by only showing a run script activity's script contents if the script run was ad-hoc.
- Stopped policy automations from running on macOS hosts until after setup experience finishes so that Fleet doesn't attempt to install software twice.
- Added tooltip informing users a test email will be sent when SMTP settings are changed.
- Added copyable SHA256 hash to the software details page.
- Added device user API error state to replace generic Fleet UI error state in Fleet desktop. 
- Revised PKG custom package parsing to pick the correct app name and bundle ID in more instances.
- Ensured consistent failing policies and total issues counts on the host details page by re-calculating these counts every time the API receives a request for that host.
- Allowed Fleet secret environment variables for the MacOS setup script.
- Validated uploaded bootstrap package to ensure that it is a Distribution package since that is required by Apple's InstallEnterpriseApplication MDM command.
- Modified the Windows MDM detection query to more accurately detect existing MDM enrollment details on hosts with multiple enrollments.
- Created consistent UI for the copy button of an input field.
- Updated the notes for the `disk_info` table to clarify usage in ChromeOS.
- Fixed an issue where the cursor on the SQL editor would sometimes become misaliged.
- Fixed slight style issues with the user menu.
- Fixed an issue where adding/updating a manual label had inconsistent results when multiple hosts shared a serial number.
- Fixed reading disk encryption key not showing up in host activities.
- Fixed a bug where a host that was wiped and re-enrolled without deleting the corresponding host row in Fleet had its old Google Chrome profiles (and other osquery-based data) showing for about an hour.
- Fixed an issue in the database migrations released in 4.68.0 where Apple devices with UDID values longer than 36 characters would cause a failure in the migration process; the `host_uuid` column for tables added by that migration has been increased to accommodate these longer UDID values.
- Fixed issue with GitOps command that prevented non-managed labels to be deleted if used by software installations.
- Fixed several corner cases with Apple DDM profile verification, including a migration to clear out "remove" operations with invalid status.
- Fixed a bug that caused a 500 error when searching for non-existent Fleet-maintained apps.
- Fixed a bug where global observers could access the "delete query" UX on the queries table.
- Fixed parsing of some MSI installer names.
- Fixed a bug where deleting an upcoming activity did not ensure the upcoming activities queue made progress in some cases.
- Fixed a CIS query (Ensure Show Full Website Address in Safari Is Enabled).

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.69.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-06-14">
<meta name="articleTitle" value="Fleet 4.69.0 | Bulk scripts improvements, Entra ID and authentik foreign vitals, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.69.0-1600x900@2x.png">
