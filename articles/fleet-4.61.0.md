# Fleet 4.61.0 | Auto-install software, email two-factor authentication (2FA), automatic Windows migration

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/f_uopfwa3ys?si=taTKh9l8iXJ-sC88" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.61.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.61.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Auto-install software
- Email two-factor authentication (2FA)
- Automatic Windows migration

### Auto-install software

IT admins can now install a Fleet-maintained app on all hosts without writing a custom policy. This simplifies software management and saves time for your end users by ensuring productivity tools like Slack and Zoom are consistently available. Learn more about automatically installing software [here](https://fleetdm.com/guides/automatic-software-install-in-fleet).

### Email two-factor authentication (2FA)

You can now enable email 2FA for Fleet user accounts. This adds an extra layer of security for your "break glass" account that's used to login to Fleet in the rare scenario that your Identify Provider (IdP) goes down. For all other accounts, the best practice is to require users to login with [single-sign on (SSO)](https://fleetdm.com/docs/deploy/single-sign-on-sso).

### Automatic Windows migration

Fleet now supports migrating Windows workstations from your old MDM solution without end user interaction. Once migrated, you can enforce [disk encryption](https://fleetdm.com/guides/enforce-disk-encryption), [OS updates](https://fleetdm.com/guides/enforce-os-updates), and other [custom OS settings](https://fleetdm.com/guides/custom-os-settings) to consolidating device management into a single, cross-platform MDM.

## Changes

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

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.61.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2024-12-17">
<meta name="articleTitle" value="Fleet 4.61.0 | Auto-install software, email two-factor authentication (2FA), automatic Windows migration">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.61.0-1600x900@2x.png">
