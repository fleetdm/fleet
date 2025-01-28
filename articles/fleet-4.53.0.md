# Fleet 4.53.0 | Better vuln matching, multi-issue hosts, & `fleetd` logs as tables.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/mqnjDNtJkjg?si=hjVjSAxTkzpTMhXD" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.53.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.53.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Improved matching to detect more vulnerabilities
* Discover multi-issue hosts
* `fleetd` logs available as tables
* End-user email support in Linux
* Improved MDM enrollment


### Improved matching to detect more vulnerabilities

Fleet has enhanced its vulnerability detection capabilities with recent updates that now include more comprehensive support for identifying vulnerabilities in the running operating system, particularly focusing on systems with custom Linux kernels. These improvements enable more precise matching and detection, helping IT and security teams better assess and mitigate risks associated with their operating environments. By expanding the scope to custom Linux kernels, Fleet addresses the unique challenges faced by environments that deviate from standard configurations, providing tailored security insights that align with the specific needs of each system. This development not only bolsters security measures but also ensures that vulnerability assessments are as accurate and actionable as possible, reflecting Fleet’s ongoing commitment to enhancing the robustness and responsiveness of its security features.


### Discover multi-issue hosts 

Administrators can now identify hosts with the most issues, providing a powerful tool for prioritizing and addressing critical problems within their fleet. This allows administrators to surface hosts with the most issues, enabling IT and security teams to quickly focus their efforts on devices requiring attention. By highlighting these problematic hosts, Fleet helps streamline the troubleshooting process, ensuring that resources are directed toward resolving the most pressing issues first. This enhances the overall efficiency of fleet management by facilitating a proactive approach to maintaining and securing the infrastructure, ultimately leading to a more stable and secure operating environment. This update aligns with Fleet's commitment to providing actionable insights and improving the effectiveness of IT operations and security measures.


### `fleetd` logs available as tables

Fleet now supports `fleetd` logs as queryable tables, giving administrators enhanced visibility and control over their fleet's operational data. This new feature allows IT and security teams to query `fleetd` logs directly, enabling detailed analysis and monitoring of the agent's activity and performance. By making these logs accessible as tables, Fleet empowers users to create custom queries to help identify patterns, diagnose issues, and ensure compliance with organizational policies. This capability enhances the transparency and manageability of fleet operations, allowing for more informed decision-making and proactive maintenance. Integrating `fleetd` logs into the query framework reflects Fleet's dedication to delivering comprehensive and actionable insights, reinforcing its commitment to improving the efficiency and effectiveness of device management and security practices.


### End-user email support in Linux

Fleet has extended its support for the `--end-user-email` flag to Linux hosts, building on a feature previously introduced for Windows. This functionality allows administrators to specify the end-user email address associated with a device directly through the `fleetctl` command line interface. By incorporating this flag, Fleet enhances its ability to link devices with their respective users accurately, facilitating improved user management and streamlined device tracking. This feature is particularly beneficial for organizations that must maintain clear records of device ownership and user assignments, ensuring that each device's usage and security policies are appropriately managed. Extending this capability to Linux hosts demonstrates Fleet's commitment to providing versatile and comprehensive solutions for multi-platform environments, empowering IT teams to maintain organized and efficient fleet operations across all major operating systems.


### Improved MDM enrollment

The `fleetd` agent will now be installed when Mobile Device Management (MDM) is enabled on a device. This ensures that organizations using MDM for their macOS devices can seamlessly integrate Fleet's monitoring and management features without disrupting existing MDM workflows. By allowing the installation of `fleetd` alongside MDM, Fleet enhances its flexibility and interoperability within diverse IT environments, providing comprehensive oversight and control over device configurations, security policies, and operational status. This feature simplifies the deployment process and ensures that all macOS devices, regardless of their management setup, can benefit from Fleet's robust capabilities. It reflects Fleet's commitment to supporting complex, multi-layered IT infrastructures and enhancing the efficiency and effectiveness of device management practices across organizations.


## Changes

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


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.53.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-06-25">
<meta name="articleTitle" value="Fleet 4.53.0 | Better vuln matching, multi-issue hosts, & `fleetd` logs as tables">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.53.0-1600x900@2x.png">
