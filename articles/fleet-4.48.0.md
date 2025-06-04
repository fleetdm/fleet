# Fleet 4.48.0 | IdP local account creation, VS Code extensions.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/zd_JFeryiQE?si=1jVm9M1YWW44uR2s" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.48.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.48.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* IdP local account creation
* Software inventory includes VS Code extensions 



### IdP local account creation

Local account creation with an Identity Provider (IdP) now prefills and locks local account creation with details sourced from the IdP. This feature streamlines the initial setup experience during the macOS out-of-the-box setup process by ensuring that the full name and account name fields are automatically populated with the user's IdP username. Additionally, it enforces password policies to be applied before the account creation is finalized, adding an extra layer of security and compliance to the enrollment process. This update simplifies the onboarding experience for end-users, allowing them to log into their Mac with their IdP credentials seamlessly. It reflects Fleet's commitment to enhancing device management efficiency and security. Fleet empowers IT administrators to ensure a consistent and secure user experience across their macOS fleet by focusing on automating and securing the setup process.


### VS Code extensions

In addressing the need for comprehensive software inventory management, Fleet has expanded its inventory capabilities to include Visual Studio Code (VS Code) extensions. This addition enables IT and security teams to gain visibility into the VS Code extensions installed across their device fleet, offering a clearer view of their environments' development tools and resources. By surfacing VS Code extensions in the software inventory, Fleet allows for a more detailed assessment of the software landscape, facilitating better compliance, security assessments, and software management practices. This feature aligns with Fleet's commitment to providing detailed and actionable insights into the software ecosystem, supporting informed decision-making and proactive management of digital assets.




## Changes


### Endpoint operations
- Added integration with Google Calendar.
  * Fleet admins can enable Google Calendar integration by using a Google service account with domain-wide delegation.
  * Calendar integration is enabled at the team level for specific team policies.
  * If the policy is failing, a calendar event will be put on the host user's calendar for the 3rd Tuesday of the month.
  * During the event, Fleet will fire a webhook. IT admins should use this webhook to trigger a script or MDM command that will remediate the issue.
- Reduced the number of 'Deadlock found' errors seen by the server when multiple hosts share the same UUID.
- Removed outdated tooltips from UI.
- Added hover states to clickable elements.
- Added cross-platform check for duplicate MDM profile names in batch set MDM profiles API.

### Device management (MDM)
- Added Windows MDM support to the `osquery-perf` host-simulation command.
- Added a missing database index to the MDM Windows enrollments table that will improve performance at scale.
- Migrate MDM-related endpoints to new paths, deprecating (but still supporting indefinitely) the old endpoints.
- Adds API functionality for creating DDM declarations, both individually and as a batch.
- Added DDM activities to the fleet UI.
- Added the `enable_release_device_manually` configuration setting for a team and no team. **Note** that the macOS automatic enrollment profile cannot set the `await_device_configured` option anymore, this setting is controlled by Fleet via the new `enable_release_device_manually` option.
- Automatically release a macOS DEP-enrolled device after enrollment commands and profiles have been delivered, unless `enable_release_device_manually` is set to `true`.

### Vulnerability management
- Added Visual Studio extensions to Fleet's software inventory.

### Bug fixes
- Fixed a bug where valid MDM enrollments would show up as unmanaged (EnrollmentState 3).
- Fixed flash message from closing when a modal closes.
- Fixed a bug where OS version information would not get detected on Windows Server 2019.
- Fixed issue where getting host details failed when attempting to read the host's BitLocker status from the datastore.
- Fixed false negative vulnerabilities on macOS Homebrew python packages.
- Fixed styling of live query disabled warning.
- Fixed issue where Windows MDM profile processing was skipping `<Add>` commands.
- Fixed UI's ability to bulk delete hosts when "All teams" is selected.
- Fixed error state rendering on the global Host status expiry settings page, fix error state alignment for tooltip-wrapper field labels across organization settings.
- Fixed `GET fleet/os_versions` and `GET fleet/os_versions/[id]` so team users no longer have access to os versions on hosts from other teams.
- `fleetctl gitops` now batch processes queries and policies.
- Fixed UI bug to render the query platform correctly for queries imported from the standard query library.
- Fixed issue where Microsoft Edge was not reporting vulnerabilities.
- Fixed a bug where all Windows MDM enrollments were detected as automatic.
- Fixed a bug where `null` or excluded `smtp_settings` caused a UI 500.
- Fixed query reports so they reset when there is a change to the selected platform or selected minimum osquery version.
- Fixed live query sort of SQL result sort for both string and numerical columns.

## Fleet 4.47.3 (Mar 26, 2024)

### Bug fixes

* Fixed a bug where valid Windows MDM enrollments would show up as unmanaged (EnrollmentState 3).

## Fleet 4.47.2 (Mar 22, 2024)

### Bug fixes

* Fixed false negative vulnerabilities on macOS Homebrew Python packages.
* Fixed policies to check "disable guest user".
* Resolved the issue where Microsoft Edge was not reporting vulnerabilities.

## Fleet 4.47.1 (Mar 18, 2024)

### Bug fixes

* Removed outdated tooltips from UI.
* Fixed an issue with Windows MDM profile processing where `<Add>` commands were being skipped.
* Team users no longer have access to OS versions on hosts from other teams for GET fleet/os_versions and GET fleet/os_versions/[id].
* Reduced the number of 'Deadlock found' errors seen by the server when multiple hosts share the same UUID.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.48.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-04-03">
<meta name="articleTitle" value="Fleet 4.48.0 | IdP local account creation, VS Code extensions.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.48.0-1600x900@2x.png">
