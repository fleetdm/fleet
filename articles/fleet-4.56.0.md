# Fleet 4.56.0 | Enhanced MDM migration, Exact CVE Search, and Self-Service VPP Apps.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/aQyePPQ0uXA?si=w9FB7AvxbOrun76O" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.56.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.56.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
* Improved end-user MDM migration
* Enforce minimum OS version for MDM enrollment
* Exact match CVE search
* Software vulnerabilities severity filter
* Self-service VPP apps
* Multiple ABM and VPP support


### Improved end-user MDM migration

Fleet has improved the end-user MDM migration workflow on macOS by enabling the migration of hosts manually enrolled in a third-party MDM over to Fleet MDM using the Fleet Desktop application. Previously, this capability was limited to hosts enrolled through Apple's Automated Device Enrollment (ADE), but with this update, manually enrolled hosts can now be seamlessly migrated to Fleet MDM. This feature is specifically available for macOS Sonoma devices (macOS 14 or greater). It makes the migration process more flexible and accessible for organizations looking to centralize their MDM management under Fleet. This enhancement simplifies the transition to Fleet MDM for a broader range of macOS devices, ensuring that all hosts can be managed consistently and securely.


### Enforce minimum OS version for MDM enrollment

Fleet now enforces a minimum operating system (OS) requirement for macOS devices before they can be enrolled into Fleet's MDM. This feature ensures that only devices running a specified minimum macOS version can be enrolled, helping organizations maintain a consistent security and compliance baseline across their fleet. By setting a minimum OS requirement, Fleet prevents older, potentially less secure macOS versions from being managed under its MDM, thereby reducing vulnerabilities and ensuring all enrolled devices meet the organization's standards. This update enhances Fleet's ability to enforce security policies from the outset, ensuring that all devices in the fleet are up-to-date and capable of supporting the latest security and management features.


### Exact match CVE search

Fleet has enhanced its CVE (Common Vulnerabilities and Exposures) search functionality by introducing exact match searching, allowing users to quickly and accurately find specific vulnerabilities across their fleet. This improvement ensures that security teams can pinpoint the exact CVE they are investigating without sifting through irrelevant results, streamlining the vulnerability management process. Additionally, Fleet provides better context in cases where no results are found, helping users understand why a particular CVE might not be present in their environment. This update improves the overall user experience in vulnerability management, making it easier to maintain security and compliance across all managed devices.


### Software vulnerabilities severity filter

Fleet has introduced improved filtering capabilities for vulnerable software, allowing users to filter vulnerabilities by severity level. This enhancement enables security teams to prioritize their response efforts by focusing on the most critical vulnerabilities, ensuring that the highest-risk issues are promptly addressed. By providing a straightforward and efficient way to filter vulnerable software based on severity, Fleet helps organizations streamline their vulnerability management processes, reducing the risk of security incidents. This update aligns with Fleet's commitment to providing powerful tools that enhance the efficiency and effectiveness of security operations across all managed devices.


### Self-Service Apple App Store apps

Fleet enables organizations to assign and install Apple App Store apps purchased through the Volume Purchase Program (VPP) directly via Self-Service using Fleet Desktop. This new feature allows IT administrators to make VPP-purchased apps available to end users seamlessly and flexibly. By integrating VPP app distribution into the Fleet Desktop Self-Service portal, organizations can streamline the deployment of essential software across their macOS devices, ensuring that users have easy access to the tools they need while maintaining control over software distribution. This update enhances the overall user experience and operational efficiency, empowering end users to install approved applications with minimal IT intervention.


### Multiple Apple Business Manager and VPP support

Fleet now enables administrators to add and manage multiple Apple Business Manager (ABM) and Volume Purchase Program (VPP) tokens within a single Fleet instance. This feature is designed for both Managed Service Providers (MSPs) and large enterprises, allowing them to create separate automatic enrollment and App Store app workflows for different clients or divisions, each with their own ABM and VPP tokens. Whether you’re managing devices for multiple customers or supporting large organizations with distinct divisions, this update simplifies the process of handling macOS, iOS, and iPadOS devices. With support for multiple ABM and VPP connections, Fleet streamlines software and device management across varied environments, providing a scalable solution for both MSPs and enterprises looking to centralize control while maintaining flexibility for different user groups.


## Changes

**NOTE:** Beginning with Fleet v4.55.0, Fleet no longer supports MySQL 5.7 because it has reached [end of life](https://mattermost.com/blog/mysql-5-7-reached-eol-upgrade-to-mysql-8-x-today/#:~:text=In%20October%202023%2C%20MySQL%205.7,to%20upgrade%20to%20MySQL%208.). The minimum version supported is MySQL 8.0.36.

## Fleet 4.56.0 (Sep 7, 2024)

### Endpoint operations

- Added index to `query_results` DB table to speed up finding last query timestamp for a given query and host.
- Added a link in the UI to the error message when a CSR can't be downloaded due to missing private key.
- Added a disabled overlay to the Other Workflows modal on the policy page.
- Improved performance of live queries to accommodate for higher volumes when utilizing zero-trust workflows.
- Improved `fleetctl` gitops error message when trying to change team name to a team that already exists.

### Device management

- Added server support for multiple VPP tokens.
- Added new endpoints and updated existing endpoints for managing multiple Apple Business Manager tokens.
- Added support for S3 to store MDM bootstrap packages (uses the same bucket configuration as for software installers).
- Added support to UI for self service VPP software.
- Added backend and gitops support for self service VPP.
- Added ability for MDM migrations if the host is manually enrolled to a 3rd party MDM.
- Added an offline screen to the macOS MDM migration flow.
- Added new ABM page to Fleet UI.
- Added new VPP page to the fleet UI
- Added support to track the Apple Business Manager "terms expired" API error per token, as well as a global flag that gets set as soon as one token has its terms expired.
- Updated the instructions on "My device" for MDM migrations on pre-Sonoma macOS hosts.
- Updated to allow multiple teams to be assigned to the same VPP Token.
- Updated process so that deleting installed software or VPP app now makes it available for re-installation.
- Updated to enforce minimum OS version settings during Apple Automated Device Enrollment (ADE).
- Updated ABM ingestion so that deleted iOS/iPadOS host will continue to report to Fleet as long as host is in Apple Business Manager (ABM).
- Updated so that refetching an offline iOS/iPadOS host will not add new MDM commands to the queue if previous refetch has not completed yet.
- Updated UI so that downloading a software installer package now shows the browser's built-in progress bar.
- Updated relevant documentation to include references to multiple ABM and VPP tokens.
- Consolidated Automatic Enrollment and VPP settings under the MDM settings integration page.
- Cleared apps associated with a VPP token if it's moved off of a team.

### Vulnerability management

- Added ALAS bulletins as vulnerability source for Amazon Linux (instead of OVAL for Amazon Linux 2, and adds support for Amazon Linux 1, 2022, and 2023).
- Added matching rules for July and August Microsoft 365 security updates (https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates).
- Added the following filters to `/software/titles` and `/software/versions` API endpoints: `exploit: bool`, `min_cvss_score: float`, `max_cvss_score: float`.
- Updated software titles/versions tables to allow for filtering by vulnerabilities including severity and known exploit.
- Updated to use empty CVE description when the NVD CVE feed doesn't include description entries (instead of panicking).
- Updated matching software that is not installed by Fleet so that it shows up as 'Available for install' on host details page.
- Updated base images of `fleetdm/fleetctl`, `fleetdm/bomutils` and `fleetdm/wix` to fix critical vulnerabilities found by Trivy.
- Updated vulnerability scanning to use `macos` SW target for CPEs of homebrew packages.
- Updated vulnerability scanning to not ignore software with non-ASCII en dash and em dash characters.
- Updated `GET /api/v1/fleet/vulnerabilities/{cve}` endpoint to add validation of CVE format, and a 204 response. The 204 response indicates that the vulnerability is known to Fleet but not present on any hosts.
- Updated the UI to add new empty states for searching vulnerabilities: invalid CVE format searched, a known CVE serached but not present on hosts, not a known CVE searched, exploited vulnerability empty state, operating systems empty state, new icons.

### Bug fixes and improvements

- Added support for MySQL 8.4.2 LTS.
- Updated Go to go1.22.6.
- Updated Fleet server to now accept arguments via stdin. This is useful for passing secrets that you don't want to expose as env vars, in the command line, or in the config file.
- Updated text for "Turn on MDM" banners in UI.
- Updated ABM host tooltip copy on the manage host page to clarify when host vitals will be available to view.
- Updated copy on auotmatic enrollment modal on my device page.
- Updated host details activities tooltip and empty state copy to reflect recently added capabilities.
- Updated Fleet Free so users see a Premium feature message when clicking to add software.
- Updated usage reporting to report statistics on new AI features, maintenance window, and `fleetd`.
- Fixed bug where configuration profile was still showing the old label name after the name was updated.
- Fixed a bug when a cached prepared statement gets deleted in the MySQL server itself without Fleet knowing.
- Fixed a bug where the wrong API path was used to download a software installer.
- Fixed the failing_host_count so it is never 0. This count is normally updated once an hour during cleanups_then_aggregation cron job.
- Fixed CVE-2024-4030 in Vulncheck feed incorrectly targeting non-Windows hosts.
- Fixed a bug where the "Self-service" filter for the list of software and the list of host's software did not take App Store apps into account.
- Fixed a bug where the "My device" page in Fleet Desktop did not show the self-service software tab when App Store apps were available as self-install.
- Fixed a bug where a software installer (a package or a VPP app) that has been installed on a host still shows up as "Available for install" and can still be requested to be installed after the host is transferred to a different team without that installer (or after the installer is deleted).
- Fixed the "Available for install" filter in the host's software page so that installers that were requested to be installed on the host (regardless of installation status) also show up in the list.
- Fixed UI popup messages bleeding off viewport in some cases.
- Fixed an issue with the scheduling of cron jobs at startup if the job has never run, which caused it to be delayed.
- Fixed UI to display the label names in case-insensitive alphabetical order.

## Fleet 4.55.2 (Sep 05, 2024)

### Bug fixes

* Removed validation of APNS certificate from server startup. This was no longer necessary because we now allow for APNS certificates to be renewed in the UI.
* Fixed logic to properly catch and log APNs errors.

## Fleet 4.55.1 (Aug 14, 2024)

### Bug fixes

* Added a disabled overlay to the Other Workflows modal on the policy page.
* Updated text for "Turn on MDM" banners in UI.
* Fixed a bug when a cached prepared statement got deleted in the MySQL server itself without Fleet knowing.
* Continued with an empty CVE description when the NVD CVE feed didn't include description entries (instead of panicking).
* Scheduled maintenance events are now scheduled over calendar events marked "Free" (not busy) in Google Calendar.
* Fixed a bug where the wrong API path was used to download a software installer.
* Improved fleetctl gitops error message when trying to change team name to a team that already exists.
* Updated ABM (Apple Business Manager) host tooltip copy on the manage host page to clarify when host vitals will be available to view.
* Added index to query_results DB table to speed up finding the last query timestamp for a given query and host.
* Displayed the label names in case-insensitive alphabetical order in the fleet UI.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.56.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-09-07">
<meta name="articleTitle" value="Fleet 4.56.0 | Enhanced MDM migration, Exact CVE Search, and Self-Service VPP Apps.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.56.0-1600x900@2x.png">
