# Fleet 4.58.0 | Run script on policy failure, Fleet-maintained apps, Sequoia firewall status.

<div purpose="embedded-content">
   <iframe src="ttps://www.youtube.com/embed/2vJsE5K4ru4?si=iKjxLYHw1PUTAdTV" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.58.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.58.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
* Policy failure: execute script
* Fleet-maintained apps for macOS
* Sequoia firewall status
* RPM package support

### Policy failure: execute script

Fleet now supports automatically running a script when a policy fails, providing a more proactive approach to policy enforcement and remediation. This feature allows administrators to define scripts that will be executed whenever a device fails to meet a specified policy, enabling automated corrective actions to be taken immediately. For example, if a security policy detects a misconfiguration or outdated software version, a script can be triggered to fix the issue or notify the user. This capability helps streamline maintaining compliance and ensures that devices are quickly brought back into alignment with organizational standards, reducing the need for manual intervention and enhancing overall fleet management efficiency.

### Fleet-maintained apps for macOS

Fleet now supports Fleet-maintained apps on macOS, making it easier for admins to deploy and manage commonly used applications across their fleet. This feature allows IT teams to quickly install and update a curated selection of essential apps maintained by Fleet, ensuring that these applications are always up-to-date and secure. By simplifying managing software on macOS devices, Fleet-maintained apps help organizations maintain consistency in their software deployments, improve security by ensuring software is current, and reduce the administrative burden of manually managing application updates. This update underscores Fleet's commitment to providing user-friendly solutions for efficient and secure device management.

### Sequoia firewall status

With macOS 15 Sequoia, the existing `alf` table in osquery no longer returns firewall status results due to changes in how firewall settings are structured in the new OS. To address this, Fleet has added support for reporting firewall status on macOS 15, ensuring administrators can monitor and manage firewall configurations across their devices. This update helps maintain visibility into critical security settings even as Apple introduces changes to macOS, allowing IT teams to ensure compliance with security policies and proactively address any firewall configuration issues. This enhancement reflects Fleet's commitment to adapting to evolving platform changes while providing robust security and monitoring capabilities across all supported devices.

### RPM package support

Fleet now supports RPM package installation on Linux distributions such as Fedora and Red Hat, significantly expanding its software management capabilities. With this enhancement, IT admins can deploy and manage RPM packages directly from Fleet, streamlining software installation, updating, and maintenance across Linux hosts. This addition enables organizations to leverage Fleet for consistent software management across a broader range of Linux environments, improving operational efficiency and simplifying package deployment workflows. By supporting RPM packages, Fleet continues to enhance its flexibility and adaptability in managing diverse device fleets.

## Changes

**NOTE:** Beginning with Fleet v4.55.0, Fleet no longer supports MySQL 5.7 because it has reached [end of life](https://mattermost.com/blog/mysql-5-7-reached-eol-upgrade-to-mysql-8-x-today/#:~:text=In%20October%202023%2C%20MySQL%205.7,to%20upgrade%20to%20MySQL%208.). The minimum version supported is MySQL 8.0.36.

**Endpoint Operations:**

* Added builtin label for Fedora Linux.  **Warning:** Migrations will fail if a pre-existing 'Fedora Linux' label exists. To resolve, delete the existing 'Fedora Linux' label.
* Added ability to trigger script run on policy failure.
* Updated GitOps script and software installer relative paths to now always relative to the file they're in. This change breaks existing YAML files that had to account for previous inconsistent behavior (e.g. script paths declared in no-team.yml being relative to default.yaml one directory up).
* Improved performance for host details and Fleet Desktop, particularly in environments using high volumes of live queries.
* Updated activity cleanup job to remove all expired live queries to improve API performance in environment using large volumes of live queries.  To note, the cleanup cron may take longer on the first run after upgrade.
* Added an event for when a policy automation triggers a script run in the activity feed.
* Added battery status to Windows host details.

**Device Management (MDM):**

* Added the `POST /software/fleet_maintained_apps` endpoint for adding Fleet-maintained apps.
* Added the `GET /software/fleet_maintained_apps/{app_id}` endpoint to retrieve details of a Fleet-maintained app.
* Added API endpoint to list team available Fleet-maintained apps.
* Added UI for managing Fleet-maintained apps.
* Updated add software modal to be seperate pages in Fleet UI.
* Added support for uploading RPM packages.
* Updated the request timeouts for software installer edits to be the same as initial software installer uploads.
* Updated UI for software uploads to include upload progress bar.
* Improved performance of SQL queries used to determine MDM profile status for Apple hosts.

**Vulnerability Management:**

* Fixed MSRC feed pulls (for NVD release builds) in environments where GitHub access is authenticated.

**Bug fixes and improvements:**

* Added the 'Unsupported screen size' UI on the My device page.
* Removed redundant built in label filter pills.
* Updated success messages for lock, unlock, and wipe commands in the UI.
* Restricted width of policy description wrappers for better UI.
* Updated host details about section to condense information into fewer columns at smaller widths.
* Hid CVSS severity column from Fleet Free software details > vulnerabilities sections.
* Updated UI to remove leading/trailing whitespace when creating or editing team or query names.
* Added UI improvements when selecting live query targets (e.g. styling, closing behavior).
* Updated API to return 409 instead of 500 when trying to delete an installer associated with a policy automation.
* Updated battery health definitions to be defined as cycle counts greater than 1000 or max capacity falling under 80% of designed capacity for macOS and Windows.
* Added information on how battery health is defined to the UI.
* Updated UI to surface duplicate label name error to user.
* Fixed software uninstaller script for `pkg`s to only remove '.app' directories installed by the package.
* Fixed "no rows" error when adding a software installer that matches an existing title's name and source but not its bundle ID.
* Fixed an issue with the migration adding support for multiple VPP tokens that would happen if a token is removed prior to upgrading Fleet.
* Fixed UI flow for observers to easily query hosts from the host details page.
* Fixed bug with label display names always sentence casing.
* Fixed a bug where a profile wouldn't be removed from a host if it was deleted or if the host was moved to another team before the profile was installed on the host.
* Fixed a bug where removing a VPP or ABM token from a GitOps YAML file would leave the team assignments unchanged.
* Fixed host software filter bug that resets dropdown filter on table changes (pagination, order by column, etc).
* Fixed UI bug: Edit team name closes modal.
* Fixed UI so that switching vulnerability search types does not cause page re-render.
* Fixed UI policy automation truncation when selecting software to auto-install.
* Fixed UI design bug where software package file name was not displayed as expected.
* Fixed a small UI bug where a button overlapped some copy.
* Fixed software icon for chrome packages.

## Fleet 4.57.3 (Oct 11, 2024)

### Bug fixes

* Fixed Orbit configuration endpoint returning 500 for Macs running Rapid Security Response macOS releases that are enrolled in OS major version enforcement.

## Fleet 4.57.2 (Oct 03, 2024)

### Bug fixes

* Fixed software uninstaller script for `pkg`s to only remove '.app' directories installed by the package.

## Fleet 4.57.1 (Oct 01, 2024)

### Bug fixes

* Improved performance of SQL queries used to determine MDM profile status for Apple hosts.
* Ensured request timeouts for software installer edits were just as high as for initial software installer uploads.
* Fixed an issue with the migration that added support for multiple VPP tokens, which would happen if a token was removed prior to upgrading Fleet.
* Fixed a "no rows" error when adding a software installer that matched an existing title's name and source but not its bundle ID.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.58.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-10-16">
<meta name="articleTitle" value="Fleet 4.58.0 | Run script on policy failure, Fleet-maintained apps, Sequoia firewall status.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.58.0-1600x900@2x.png">
