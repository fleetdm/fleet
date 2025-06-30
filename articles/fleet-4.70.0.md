# Fleet 4.70.0 | Entra ID conditional access, Android work profiles, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/HxBQvlV14Lc?si=VLYS7QxPuP3TLbjG" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.70.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.70.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Entra ID conditional access
- One-time code for custom SCEP certificate authorities (CAs)
- Work profiles for personal (BYOD) Android
- Script reports
- Teams search

### Entra ID conditional access

Fleet now supports [Microsoft Entra ID for conditional access](https://fleetdm.com/guides/entra-conditional-access-integration). This allows IT and Security teams to block third-party app logins when a host is failing one or more policies.

### One-time code for custom SCEP certificate authorities (CAs)

Fleet now supports one-time code verification when requesting certificates from a custom SCEP certificate authority (CA). This adds a layer of security to ensure only hosts enrolled to Fleet can request certificates that [grant access to corporate resouces (Wi-Fi or VPNs)](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Work profiles for personal (BYOD) Android

Fleet has removed the Android MDM feature flag. IT admins can now [enroll BYOD Android hosts](https://fleetdm.com/guides/android-mdm-setup#basic-article) and see host vitals. Support for OS updates, configuration profiles, and more coming soon.

### Script reports

Fleet users can now see which hosts successfully ran a script, which errored, and which are still pending. This helps with troubleshooting and ensures scripts reach all intended hosts. Learn more about running scripts in Fleet in the [scripts guide](https://fleetdm.com/guides/scripts).

### Teams search

Users managing many [teams in Fleet](https://fleetdm.com/guides/teams) can now search in the teams dropdown. This makes it faster to navigate and switch between teams in Fleet's UI.

## Changes

### Security Engineers
- Updated vulnerabilities feed to fall back to non-primary CVSSv2/v3 sources when primary (NVD) data is not available, instead of omitting scores entirely.
- Updated custom SCEP proxy implementation to include one-time challenges.
- Added the `source` and `username` fields for host certificates, reporting 'system' or 'user' based on which keychain it was from (for `macOS`, it will be 'user' if coming from the "login" keychain), and the corresponding `username` if the source is 'user'.
- Updated certificates card on the host details and my device page to show a new keychain column.

### IT Admins
- Enabled Android MDM support. The functionality is limited to turning on Android MDM and enrolling a BYOD device. 
> **NOTE:** If your server was already using Android via the experimental DEV_ANDROID_ENABLED=1 flag, please turn off Android MDM before updating your Fleet server.
- Added support for filtering the hosts page for hosts with any of the 3 batch script execution statuses.
- Extended `POST /api/v1/fleet/hosts/:id/wipe` endpoint to allow users to specify the type of remote wipe for windows hosts.
- Improved releasing a macOS device during ADE enrollment, by increasing the frequency of checks for readiness.
- Added an audit log activity item for automatic install policy creation.

### Other improvements and bug fixes
- Updated the Open Policy Agent (OPA) dependency to v1.4.2. 
> **NOTE**: This upgrade drops support for YAML 1.1 in configuration files. If you use the `-c` option to specify a configuration file when starting the Fleet server, you will need to update any `yes` or `on` values in the file to `true`, and any `no` or `off` values to `false`.
- Improved error and loading state for self-service page.
* Implemented searching the teams dropdown.
- Removed sort column buttons for host software columns that do not support sorting.
- Updated migrations to use the `utf8mb4_unicode_ci` collation across all tables and added a test to validate that new migrations use this collation.
- Added new optional parameter `--outfile` to fleetctl package to override the filename being generated.
- Updated software detection so that a new installer uploaded over an FMA app does not report as an FMA app. 
- Improved error when trying to apply builtin labels.
- Updated copy and remove platform callout in manage automations modal.
- Update UI references to "Frequency" to now say "Interval".
- Prevented editing the UI MDM > End user migration section when GitOps mode is enabled, since this is GitOps-configurable.
- Made the gap between characters in password fields consistent.
- Updated to consistent 14px font size across all input and dropdown fields.
- Removed username requirements for certain MDM CIS policies.
- Added macOS redis cluster support.
- Changed to using DeleteObject S3 api for GCP interoperability.
- Updated to use the Source Code Pro font in the Disk encryption key modal for clear differentiation betweenvthe letter oh and the number zero.
- Updated go to 1.24.4
- Fixed result count shown when running a policy.
- Fixed bug with the 'Observers can run this query' tooltip due to missing styling rules.
- Fixed possible user invite race condition.
- Fixed issue where NDES SCEP admin page was parsed using wrong UTF16 endianness.
- Fixed manual labels in gitops not selecting hosts by hardware serial or uuid.
- Fixed a database bug where the `host_uuid` column was too small in some secondary tables related to ADE-enrollment and IdP accounts.
- Fixed missing CORS header check for JSON requests.
- Fixed bug when listing software titles for 'All teams' which caused duplicated entries.
- Fixed a bug that caused custom OS settings targeted using "include any" label rules to never verify on hosts that only included a subset of the targeted labels
- Fixed the Docker Fleet-maintained app install script to prevent a successful install from showing
up as a failure due to directory existence checks (live as of 2025-06-13 FMA update).
- Fixed issue causing a 500 error when clicking "Manage Automations" from the Queries page when osquery logging has certain configurations.
- Fixed issue where you could not delete a bootstrap package.
- Fixed policy autofill using incorrect media-type for query.
- Fleet Free: Removed the installer dropdown (Premium-only) from the Software page and Host details > Software tab as installer filtering isn’t applicable on the Free tier.
- Fixed issue where users were not able to reenable end user migration in the UI.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.70.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-06-30">
<meta name="articleTitle" value="Fleet 4.70.0 | Entra ID conditional access, Android work profiles, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.70.0-1600x900@2x.png">
