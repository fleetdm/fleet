# Fleet 4.44.0 | Script execution, host expiry, and host targeting improvements.

![Fleet 4.44.0](../website/assets/images/articles/fleet-4.44.0-1600x900@2x.png)

Fleet 4.44.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.44.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Run scripts on online/offline hosts
* Label-based profile enablement
* Per-team host expiry
* Enroll secret moves to Keychain and Credentials Manager


### Run scripts on online/offline hosts

Fleet now allows IT administrators to execute scripts on hosts, irrespective of their online or offline status. This enhancement allows for a more flexible script execution process, catering to various operational scenarios. Administrators can now schedule and run scripts on any host, regardless of connectivity status, and track the script's execution.

Additionally, this feature provides a comprehensive view of past and upcoming activities related to script execution for a host. IT admins can see a chronological list of actions, including both executed and scheduled scripts, offering clear visibility into the timing and sequence of these activities. This capability is particularly beneficial for ensuring that essential scripts are run in an orderly and timely manner, enhancing the overall management and maintenance of the fleet.


### Label-based profile enablement

IT administrators can now activate profiles for hosts based on specific labels, enabling more dynamic and attribute-based profile management. This functionality is particularly useful for tailoring configurations and policies to hosts that meet certain criteria, such as operating system versions. For example, an IT admin can now set a profile only to be applied to macOS hosts at or above macOS version 13.3. This approach facilitates a more granular and efficient management of host settings, ensuring that profiles are applied in a manner that aligns with each host's characteristics and requirements while also maintaining a consistent baseline across the fleet.


### Per-team host expiry

Host expiry settings can now be customized for each team. This feature addresses the diverse requirements of different groups of devices within an organization, such as servers and workstations. With this new functionality, endpoint engineers can set varied expiry durations based on the specific needs of each team. For instance, a shorter expiry period, like 1 day, can be configured for teams of servers, whereas a longer duration, such as 30 days, can be applied to your workstation teams. This flexibility ensures that each team's expiry settings are tailored to their operational tempo and requirements, providing a more efficient and effective management of device lifecycles within Fleet.


### Enroll secret moves to Keychain and Credentials Manager

Fleet's latest update addresses a crucial security concern by altering how the `fleetd` enroll secret is stored on macOS and Windows hosts. In response to the need for heightened security measures, `fleetd` will now store the enroll secret in Keychain Access on macOS hosts and in Credentials Manager on Windows hosts rather than on the filesystem. This change significantly enhances security by safeguarding the enroll secret from unauthorized access, thus preventing bad actors from enrolling unauthorized hosts into Fleet.

This update includes a migration process for existing macOS and Windows installations where the enroll secret will be moved from the filesystem to the respective secure storage systems - Keychain Access for macOS and Credentials Manager for Windows. However, Linux hosts will continue to store the enroll secret on the filesystem. This improvement demonstrates Fleet's commitment to providing robust security features, ensuring that sensitive information like enroll secrets is securely managed and less susceptible to unauthorized access.




## Changes

* **Endpoint operations**:
  - Removed rate-limiting from `/api/fleet/orbit/ping` and `/api/fleet/device/ping` endpoints.
  - For Windows hosts, fleetd now uses Windows Credential Manager for enroll secret.
  - For macOS hosts, fleetd stores and retrieves enroll secret from macOS keychain for non-MDM flow.
  - Query reports feature now supports a custom `pack_delimiter` in agent settings.
  - Packaged `fleetctl` for macOS as a universal binary (native support for both amd64 and arm64 architectures).
  - Added new flow for `fleetctl package --type=msi` on macOS using arm64 processor.
  - Teams can now configure their own host expiry settings.
  - Added UI for host details activity card.
  - Added `host_count_updated_at` to policy API responses.
  - Added "Run script" action to host details page.
  - Created the "script ran" activity linked to its host.
  - Updated host details page and `GET /api/v1/fleet/hosts/:id` endpoint so that failing policies are listed first.

* **Device management (MDM)**:
  - Added new endpoints `GET /api/v1/fleet/mdm/manual_enrollment_profile` and scripts related endpoints (`/hosts/:id/activity`, `/hosts/:id/activity/upcoming`).
  - Added support for label-based MDM profiles reconciliation.
  - Improved MDM migration puppet module.
  - Added Windows scripts for MDM unenrollment and fleetd removal.
  - Added the profile's `labels` object to MDM profiles response payload.
  - Updated UI with ability to target MDM profiles by label.
  - Added ability to configure custom `configuration_web_url` values in DEP profile.
  - Fixed a bug causing MDM SSO to fail with certain configurations.
  - Fixed queries reporting inconsistent MDM enrollment status in Windows.

* **Vulnerability management**:
  - Added support for detecting operating system vulnerabilities for macOS and Windows.
  - Corrected Windows OS false negative for multiple OS build remediations.
  - Fixed issue with incorrect `resolved_in_version` for vulnerabilities.

### Bug fixes and improvements

  - Added "No report" text for query results not saved in Fleet.
  - Updated forms across the UI for consistent styling.
  - Improved UX for globally enabling/disabling SSO.
  - Added new consistent header styling across the app.
  - Clearer browser page titles and CTAs for Observer+.
  - Updated logging destination failure response to return a 4xx error instead of 500.
  - Addressed issues with query reports and host expiry settings.
  - Resolved platform compatibility checker issues with deprecated osquery tables.
  - Updated Go to version 1.21.6.
  - osquery flag validation updated for osquery 5.11.
  - Fixed validation and error handling for `/api/fleet/orbit/device_token` and other endpoints.
  - Fixed UI bugs in script functionality, side navigation content headers, and premium message alignment.
  - Fixed a bug in searching for hosts by email addresses.
  - Fixed issues with sticky errors in fleetd-chrome after querying privacy_preferences table.
  - Fixed a bug where Munki issues section was incorrectly displayed.
  - Fixed OS compatibility calculation for certain queries.
  - Fixed a bug where capital characters would not match labels containing them.
  - Fixed bug in manage hosts UI where changing the dropdown filter did not clear OS settings filter.
  - Fixed a bug in `fleetctl` where `--context` and `--debug` flags were not allowed after certain commands.
  - Fixed a bug where the UUID for Windows updates profiles was missing the `"w"` prefix.
  - Fixed a UI bug on the controls page in team targeting forms.
  - Fixed a bug where policy automations when saved were resetting automations on other pages.

## Fleet 4.43.3 (Jan 23, 2024)

### Bug fixes

* Fixed incorrect padding on the my device page.

## Fleet 4.43.2 (Jan 22, 2024)

### Bug fixes

* Improved HTTP client used by `fleetctl` and `fleetd` to prevent errors for 204 responses.
* Added free tier UI state to OS updates and setup experience pages.
* Added warning/info messages when downgrading/upgrading `fleetd` or OSQuery.
* Updated links to an expired osquery Slack invitation to go to the support page on the Fleet website.
* Cleaned settings styling.
* Created consistent loading states when using search filter.
* Fixed center styling for empty states. For `software/titles` and `software/versions` endpoints, the
  `browser` property is no longer included in the response when empty.
* Fixed the Windows MDM polling interval so that enrolled devices check-in regularly with Fleet to look for pending MDM-related actions.
* Fixed missing empty members SVG by fixing SVG IDs.
* Fixed a bug that caused the software/titles page to error.
* Fixed 2 vulnerability false positives on Microsoft Teams on MacOS.
* Fixed bug in CIS policy: Ensure an Inactivity Interval of 20 Minutes Or Less for the Screen Saver Is Enabled.

## Fleet 4.43.1 (Jan 15, 2024)

### Bug fixes

* Fixed bug where script results would sometimes show the wrong error message when a user attempts
  to run a script on a host that has scripts disabled.
* Fixed an issue with SCEP endpoints sending back 500 status codes. Should return 400 now if bad
  data is sent to SCEP API.
* Fixed text and icon alignment UI bug.
* Fixed message for script execution timeout.
* Fixed failed scripts showing the wrong error.



## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.44.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-02-05">
<meta name="articleTitle" value="Fleet 4.44.0 | Script execution, host expiry, and host targeting improvements.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.44.0-1600x900@2x.png">
