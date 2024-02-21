# Fleet 4.45.0 | Remote lock, Linux script library, osquery storage location.

![Fleet 4.45.0](../website/assets/images/articles/fleet-4.45.0-1600x900@2x.png)

Fleet 4.45.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.45.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Remote lock for macOS, Windows, and Linux
* Linux script library
* Customizable osquery data storage location


### Remote lock for macOS, Windows, and Linux

Fleet expands its device management capabilities with remote lock functionalities for macOS, Windows, and Linux systems. This development allows administrators to enhance security protocols and respond swiftly to potential security breaches by either locking a device remotely. This feature is particularly crucial in scenarios involving lost or stolen devices or when a device is suspected to be compromised. By integrating these remote actions, Fleet empowers IT and security teams with robust tools to protect organizational data and maintain device security. This update aligns with Fleet's values of ownership and results, as it offers users more control over their device fleet while ensuring effective response measures are in place for critical security incidents.


### Linux script library

A script library specifically designed for Linux hosts has been added. This complements Fleet's existing script execution functionalities and script libraries for macOS and Windows. The script library for Linux allows administrators to store, manage, and execute scripts efficiently using the Fleet UI or API, facilitating streamlined operations and maintenance tasks on Linux-based systems. This addition underscores Fleet's commitment to adaptability and inclusiveness, ensuring users can leverage the platform's full potential regardless of their operating system environment. By providing a dedicated script library for Linux, Fleet reinforces its dedication to delivering versatile and user-centric solutions that cater to the diverse needs of IT and security professionals.


### Customizable osquery data storage location

Fleet introduces a new `--osquery-db` flag to the `fleetctl` package command, catering to a unique requirement for virtual machine (VM) environments. This feature allows users to specify or update the osquery database directory for `fleetd` at the time of packaging or through an environment variable. By enabling the customization of the osquery data storage location, users can direct `fleetd` to utilize directories with more available space, optimizing resource use in VM setups. This enhancement demonstrates Fleet's commitment to ownership by giving users greater control over their Fleet configuration and results and facilitating more efficient data management in resource-constrained environments.



## Changes

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


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.45.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-02-21">
<meta name="articleTitle" value="Fleet 4.45.0 | Remote lock, Linux script library, osquery storage location.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.45.0-1600x900@2x.png">
