# Fleet 4.79.0 | macOS updates, MDM command history, Android software config, and more...

Fleet 4.79.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.79.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Latest macOS only at enrollment
- MDM command history
- Android software configuration
- Cross-platform certificate deployment

### Latest macOS only at enrollment

You can now choose to enforce the latest macOS version only for newly enrolling hosts. Already enrolled hosts are unaffected. This gives you room to manage updates with tools like Nudge instead.

### MDM command history

On the **Host details** page for macOS, iOS, and iPadOS hosts, you’ll now see a list of all MDM commands: past and upcoming. This includes both Fleet-initiated commands and those triggered by IT admins. This can help you and your support team troubleshoot why hosts are missing configuration profiles or certificates.

### Android software configuration

IT Admins can now configure Android software (`managedConfiguration`) directly in the Fleet UI, via GitOps (YAML), or via Fleet's API. This makes it easier to customize behavior across your Android hosts. For example, you can configure the default `portal` for Palo Alto's [GlobalProtect app](https://docs.paloaltonetworks.com/globalprotect/administration/globalprotect-apps/deploy-the-globalprotect-app-on-mobiles/manage-the-globalprotect-app-using-other-third-party-mdms/configure-the-globalprotect-app-for-android) to automatically navigate end users to your VPN. Learn more in Fleet's [GitOps reference docs](https://fleetdm.com/docs/configuration/yaml-files#app-store-apps).

### Cross-platform certificate deployment

You can now install certificates from any [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificate authority on corporate Android devices. This helps you connect your end users to Wi-Fi, VPN, and other tools.

This means you can now install certificates on Android, macOS, Windows, Linux, and iOS/iPadOS hosts. See all certificate authorities supported by Fleet in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

## Changes

### IT Admins
- Added ability to view past and upcoming MDM commands for a host in Fleet.
- Added ability to apply Android app configurations.
- Added support for resending Windows MDM profiles.
- Added support for renewal of custom SCEP profiles for Windows.
- Added support for team-specific labels. Currently team-specific labels must be created via spec endpoints, used by GitOps.
- Implemented ability to create, list, and delete Android certs from the UI.
- Added Android agent application (automatically deployed via Android MDM) to support automated installation of SCEP certificates on Android hosts.
- Added messaging around Apple VPP update failures due to the application being open.
- Added ability to indicate that new MacOS hosts enrolling via ADE should be updated to the latest operating system version.
- Added ability to edit Android software config in UI.

### Security Engineers
- Added support for ingesting Windows certificates via osquery.
- Added activities for when certificates templates are created/deleted.

### Other improvements and bug fixes
- Implemented streaming for the `GET /hosts` ("list hosts") API to improve performance.
- Updated API and GitOps to support `AppleOSUpdateSettings.UpdateNewHosts`.
- Added ability to search teams in dropdown when transferring teams.
- Added pagination metadata to the `GET /mdm/commands` endpoint.
- Updated the `refresh_vpp_app_versions` cron job to only attempt to refresh versions for Apple app store apps.
- Improved edit VPP UX by disabling a form that hasn't been edited.
- Updated logic used for determining whether to update a macOS host during DEP enrollment based solely on UpdateNewHosts flag.
- Added note to descriptions on schema tables using "count" as column name.
- Aligned Android MDM unenrollment endpoint with the already existing endpoint, `DELETE /api/latest/fleet/hosts/{id}/mdm`, for consistency across MDM platforms.
- Added migration for adding `update_new_hosts` flag to both App and Team configs.
- Changed the host details page to hide builtin labels in-line with other areas such as the label filter.
- Changed iOS/iPadOS and Android enrollment links on Add hosts modal to monospaced font to improve readability.
- Improved software upload progress modal.
- Improved consistency of `gitops` output language.
- Added loading state to turn off Android modal UI.
- Updated the `migrate_to_per_host_policy` cron job to no-op if Android MDM is not enabled.
- Updated software table so that all teams selection will now remove any unsupported url params.
- Improved unclear error message when uploading an APNS certificate if the CSR was not downloaded.
- Refactored RDS IAM authentication logic into a dedicated `rdsauth` package.
- Modified the automatic enrollment profile verification logic to only verify with Apple when a profile changes
- Updated S3 username/password when running in dev mode to remove outdated mentions of MinIO.
- Hid option to transfer hosts to their current team.
- Updated setup experience links to point to add software page relevant to platform.
- Revised auth requirements for /debug endpoints.
- Added additional validation to URL parameter for MS MDM auth endpoint.
- Improved SOAP message validation on Windows MDM endpoints.
- Fixed host query report to display "Report clipped" when a query has reached the 1k result limit.
- Fixed UI error message regarding adding software to a team with a duplicate title.
- Fixed an issue where batch uploading .mobileconfig profiles failed due to display name checks.
- Fixed an issue where certificate details modal overflowed the screen.
- Fixed click area of edit software file button.
- Fixed an issue where GitOps would fail if `$FLEET_SECRET` contained XML characters in XML files, due to not escaping the value.
- Fixed query behind `fleetctl get mdm-commands` to correctly get completed Windows MDM commands.
- Fixed MDM install command output to correctly display UTF-8 characters in the UI.
- Fixed missing upgrade code persistence when adding Windows software to Fleet via GitOps.
- Fixed duplicate entry error when updating upgrade_code during software ingestion.
- Fixed case sensitivity mismatches causing duplicate titles during software ingestion.
- Fixed a bug where iOS and iPadOS hosts enrolling via ABM MDM Migration did not have VPP apps installed.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.79.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-01-14">
<meta name="articleTitle" value="Fleet 4.79.0 | macOS updates, MDM command history, Android software config, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.79.0-1600x900@2x.png">
