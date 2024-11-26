# Fleet 4.59.0 | Install apps during new Mac boot, connect end users to Wi-Fi, custom URL for Apple MDM

![Fleet 4.59.0](../website/assets/images/articles/fleet-4.59.0-1600x900@2x.png)

Fleet 4.59.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.59.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Install apps and run scripts during new Mac boot
- Automatically connect end users to Wi-Fi
- Custom URL for Apple MDM

### Install apps during new Mac boot

Using Fleet, you can now block a user’s screen while software installs or scripts run during macOS Setup Assistant. This prevents users from accessing the desktop before required configurations are enforced, improving security and guaranteeing that all workstations meet organizational standards before use. Learn more in the guide [here](https://fleetdm.com/guides/macos-setup-experience).

### Automatically connect end users to Wi-Fi

With Fleet, you can now install a SCEP certificate from NDES on all macOS hosts as part of the Wi-Fi/Ethernet configuration profile. This ensures seamless and secure network access for end users. Learn more in the guide [here](https://fleetdm.com/guides/ndes-scep-proxy).

### Custom URL for Apple MDM

Fleet now provides the ability to set an alternative MDM URL to help organizations differentiate MDM traffic from other Fleet traffic, allowing the application of network rules specific to MDM communications. Learn more in the guide [here](https://fleetdm.com/guides/alternate-apple-mdm-url).

## Changes

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

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.59.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2024-11-12">
<meta name="articleTitle" value="Fleet 4.59.0 | Install apps during new Mac boot, connect end users to Wi-Fi">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.59.0-1600x900@2x.png">
