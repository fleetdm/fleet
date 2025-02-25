# Fleet 4.55.0 | MySQL 8, arm64 support, FileVault improvements, VPP support.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/tpXTJ2RX0wA?si=rOXdjGUX8dddnAmc" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.55.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.55.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* MySQL 8 support, MySQL 5.7 sunsets
* FileVault key rotation with Escrow Buddy
* FileVault enforcement at enrollment
* Arm64 support
* VPP app support for macOS
* "No team" software support

### MySQL 8 support, MySQL 5.7 sunsets

Fleet has updated its database compatibility by adding support for MySQL 8, while simultaneously dropping support for MySQL 5.7. This change aligns Fleet with the latest advancements in database technology, offering enhanced performance, security, and features available in MySQL 8. Organizations using Fleet are encouraged to upgrade their database systems to MySQL 8 to take full advantage of these improvements. By focusing on the latest supported versions, Fleet ensures that its platform remains robust, secure, and well-equipped to handle the demands of modern IT environments while phasing out older versions that may not provide the same level of performance or security.

### FileVault key rotation with Escrow Buddy

Fleet now includes support for FileVault key rotation using [Escrow Buddy](https://github.com/macadmins/escrow-buddy), a tool developed by the Netflix Client Systems Engineering team for the MacAdmins community to securely manage and rotate FileVault recovery keys on macOS devices. This feature allows IT administrators to automate the process of rotating FileVault keys, ensuring that encrypted macOS hosts remain secure while maintaining access control. By integrating with Escrow Buddy, Fleet enables seamless key management, reducing the administrative burden of manually rotating keys and enhancing the overall security posture of macOS environments. This update reflects Fleet's commitment to providing robust security tools that integrate with trusted community resources, ensuring organizations can efficiently manage device encryption and recovery processes.

### FileVault enforcement at enrollment

Fleet now supports enforcing FileVault encryption during the enrollment process for macOS devices, ensuring that all newly enrolled Macs are automatically encrypted. This feature enhances security by mandating that FileVault is enabled as part of the initial device setup, reducing the risk of unencrypted data on managed endpoints. By integrating FileVault enforcement into the enrollment workflow, Fleet helps organizations maintain a consistent security posture across their macOS fleet, ensuring compliance with internal policies and regulatory requirements. This update underscores Fleet's commitment to providing comprehensive security management tools that protect sensitive data and simplify the administration of macOS devices.

### Arm64 support

Fleet now includes support for Linux hosts running on the arm64 architecture. This update enables organizations to integrate a broader range of devices into their Fleet management system, ensuring comprehensive oversight and control across diverse hardware environments. By supporting arm64 Linux hosts, Fleet caters to the growing use of ARM-based systems in various sectors, allowing IT administrators to manage these devices with the same level of detail and efficiency as traditional x86-based hosts. This aligns with Fleet's commitment to providing versatile and inclusive device management solutions, empowering users to maintain a unified and efficient IT infrastructure.

### VPP app support for macOS

Fleet now supports installing Volume Purchase Program (VPP) apps from the Apple App Store on macOS devices. This feature enables IT administrators to deploy and manage apps purchased through Apple's VPP directly to macOS hosts, streamlining the process of distributing essential software across the organization. By integrating VPP app installations into Fleet, organizations can ensure that licensed applications are efficiently deployed to the appropriate devices, improving software management and compliance. This update enhances Fleet's capabilities in managing macOS environments, offering a more seamless and centralized approach to app distribution for enterprise and educational settings.

### "No team" software support

Fleet now supports adding software to the "No team" team, providing greater flexibility in managing software across an organization's devices. This feature allows administrators to deploy and manage software that applies universally without being restricted to specific teams. By adding software to the "No team" team, IT teams can ensure that essential tools and applications are available across all devices, regardless of their team assignment. This update simplifies the management of widely used software and enhances the ability to maintain consistency and compliance across the entire fleet. It reflects Fleet's commitment to offering versatile solutions that cater to diverse organizational needs and streamline device management processes.

## Changes

**NOTE:** Beginning with v4.55.0, Fleet no longer supports MySQL 5.7 because it has reached [end of life](https://mattermost.com/blog/mysql-5-7-reached-eol-upgrade-to-mysql-8-x-today/#:~:text=In%20October%202023%2C%20MySQL%205.7,to%20upgrade%20to%20MySQL%208.). The minimum version supported is MySQL 8.0.36.

### Endpoint Operations

- Added support for generating `fleetd` packages for Linux ARM64.
- Added new `fleetctl package` --arch flag.
- Updated `fleetctl package` command to remove the `--version` flag. The version of the package can be controlled by `--orbit-channel` flag.
- Updated maintenance window descriptions to update regularly to match the failing policy description/resolution.
- Updated maintenance windows using Google Calendar so that calendar events are now recreated within 30 seconds if deleted or moved to the past.
  - Fleet server watches for potential changes for up to 1 week after original event time. If event is moved forward more than 1 week, then after 1 week Fleet server will check for event changes once every 30 minutes.
  - **NOTE:** These near real-time updates may add additional load to the Google Calendar API, so it is recommended to use API usage alerts or other monitoring methods.

### Device Management

- Integrated [Escrow Buddy](https://github.com/macadmins/escrow-buddy) to add enforcement of FileVault during the MacOS Setup Assistant process for hosts that are 
enrolled into teams (or no team) with disk encryption turned on. Thank you homebysix and team!
- Added OS updates support to iOS/iPadOS devices.
- Added iOS and iPadOS device details refetch triggered with the existing `POST /api/latest/fleet/hosts/:id/refetch` endpoint.
- Added iOS and iPadOS user-installed apps to Fleet.
- Added iOS and iPadOS apps to be installed using Apple's VPP (Volume Purchase Program) to Fleet.
- Added support for VPP to GitOps.
- Added the `POST /mdm/apple/vpp_token`, `DELETE /mdm/apple/vpp_token` and `GET /vpp` endpoints and related functionality.
- Added new `GET /software/app_store_apps` and `POST /software/app_store_apps` endpoints and associated functionality.
- Added the associated VPP apps to the `GET /software/titles` and `GET /software/titles/:id` endpoints.
- Added the associated VPP apps to the `GET /hosts/:id/software` and `GET /device/:token/software` endpoints.
- Added support to delete a VPP app from a team in `DELETE /software/titles/:software_title_id/available_for_install`.
- Added `exclude_software` query parameter to "Get host by identifier" API.
- Added ability to add/remove/disable apps with VPP in the Fleet UI.
- Added a warning banner to the UI if the uploaded VPP token is about to expire/has expired.
- Added UI updates for VPP feature on host software and my device pages.
- Added global activity support for VPP-related activities.
- Added UI features for managing VPP apps for iPadOS and iOS hosts.
- Updated profile activities to include iOS and iPadOS.
- Updated Fleet UI to show OS version compliance on host details page.
- Added support for "No teams" on all software pages including adding software installers.
- Added DB migration to support VPP software features.
- Added DB migration to migrate older team configurations to the new version that includes both installers and App Store apps.
- Linux lock/unlock scripts now make use of pam_nologin to keep AD users locked out.
- Installed software list now includes Linux .deb packages that are 'on hold'.
- Added a special-case to properly name the Notion .exe Windows installer the same as how it will be reported by osquery post-install.
- Increased threshold to renew Apple SCEP certificates for MDM enrollments to 180 days.

### Vulnerability Management

- Fixed CVEs identified as 'Rejected' in NVD not matching against software.
- Fixed false negative vulnerabilities with IntelliJ IDEA CE and PyCharm CE installed via Homebrew.

### Bug fixes and improvements

- Dropped support for MySQL 5.7 and raised minimum required to MySQL 8.0.36.
- Updated software pre-install to use new GitOps format for query.
- Updated UI tooltips for pending OS settings.
- Added a migration to migrate older team configurations to the new version that includes both installers and App Store apps.
- Fixed a styling issue in the controls > OS settings > disk encryption table.
- Fixed a bug in `fleetctl preview` that was causing it to fail if Docker was installed without support for the deprecated `docker-compose` CLI.
- Fixed an issue where the app-wide warning banners were not showing on the initial page load.
- Fixed a bug where the hosts page would sometimes allow excess pagination.
- Fixed a bug where software install results could not be retrieved for deleted hosts in the activity feed.
- Fixed path that was incorrect for the download software installer package endpoint `GET /software/titles/:software_title_id/package`.
- Fixed a bug that set `last_enrolled_at` during orbit re-enrollment, which caused osquery enroll failures when `FLEET_OSQUERY_ENROLL_COOLDOWN` is set.
- Fixed the "Available for install" filter in the host's software page so that installers that were requested to be installed on the host (regardless of installation status) also show up in the list.
- Fixed a bug where Fleet google calendar events generated by Fleet <= 4.53.0 were not correctly processed by 4.54.0.
- Fixed a bug in `fleetctl preview` that was causing it to fail if Docker was installed without support for the deprecated `docker-compose` CLI.
- Fixed a bug where software install results could not be retrieved for deleted hosts in the activity feed.
- Fixed a bug where a software installer (a package or a VPP app) that has been installed on a host still shows up as "Available for install" and can still be requested to be installed after the host is transferred to a different team without that installer (or after the installer is deleted).
- Fixed the "Available for install" filter in the host's software page so that installers that were requested to be installed on the host (regardless of installation status) also show up in the list.

## Fleet 4.54.1 (Jul 24, 2024)

### Bug fixes
- Fixed a startup bug by performing an early restart of orbit if an agent options setting has changed.
- Implemented a small refactor of orbit subsystems.
- Removed the `--version` flag from the `fleetctl package` command. The version of the package can now be controlled by the `--orbit-channel` flag.
- Fixed a bug that set `last_enrolled_at` during orbit re-enrollment, which caused osquery enroll failures when `FLEET_OSQUERY_ENROLL_COOLDOWN` is set .
- In `fleetctl package` command, removed the `--version` flag. The version of the package can be controlled by `--orbit-channel` flag.
- Fixed a bug where Fleet google calendar events generated by Fleet <= 4.53.0 were not correctly processed by 4.54.0.
- Re-enabled cached logins after windows Unlock.



## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.55.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-08-09">
<meta name="articleTitle" value="Fleet 4.55.0 | MySQL 8, arm64 support, FileVault improvements, VPP support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.55.0-1600x900@2x.png">
