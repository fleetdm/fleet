# Fleet 4.54.0 | Maintenance window improvements, target hosts via label exclusion.

![Fleet 4.54.0](../website/assets/images/articles/fleet-4.54.0-1600x900@2x.png)

Fleet 4.54.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.54.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Maintenance window improvements
* Target hosts via label exclusion


### Maintenance window improvements

Fleet continues to enhance our _[maintenance windows](https://fleetdm.com/announcements/fleet-in-your-calendar-introducing-maintenance-windows)_ feature (a.k.a. Fleet in your calendar) by introducing several updates to improve the scheduling and visibility of maintenance activities. One of the notable updates includes the ability to store and display the next scheduled maintenance window for each host directly on the host details page. This feature gives administrators an immediate view of upcoming maintenance tasks, ensuring they can effectively plan and manage their operations. Additionally, Fleet now allows the creation of new maintenance windows in real time, giving IT teams the flexibility to respond to urgent maintenance needs without waiting for pre-scheduled intervals.

Further enhancing the clarity and utility of maintenance scheduling, Fleet has updated the maintenance calendar events to include the host's display name in the event body. This makes it easier for administrators to identify which devices are affected by scheduled maintenance at a glance. Moreover, a new scheduling rule has been implemented where new maintenance windows are created the following Tuesday, providing a predictable and consistent timeline for regular maintenance activities. These improvements underscore Fleet's commitment to providing efficient, user-friendly tools that support proactive device management and operational planning, ensuring that maintenance tasks are seamlessly integrated into the organization's workflow.


### Target hosts via label exclusion

Fleet has enhanced its targeting capabilities by adding support for excluding specific labels when managing and deploying configuration profiles to hosts. This feature allows administrators to precisely control which devices are affected by particular settings or policies by excluding hosts that match specified labels. For instance, if an organization has a group of conference room computers that should not receive a particular configuration, administrators can now easily exclude these devices by applying the relevant label exclusions. This added granularity ensures more accurate and tailored management of devices, reducing the risk of unintended changes and enhancing overall operational efficiency. By allowing the exclusion of any label when targeting hosts, Fleet demonstrates its commitment to providing flexible, user-centric solutions that cater to the nuanced needs of modern IT environments.


## Changes

### Endpoint Operations

- Updated scheduled maintenance windows so that calendar events are now recreated within 30 seconds if deleted or moved to the past.
- Updated scheduled maintenance to weekly on Tuesdays (previously monthly on the third Tuesday of the month).
- Added a host's upcoming scheduled maintenance window, if any, on the host details page of the UI and in host responses from the API.
- Updated `fleetctl gitops` to be used to rename teams.
  - **NOTE:** `fleetctl gitops` needs to have previously run with this Fleet/fleetctl version or later.
  - The team name is changed if the YAML config is applied from the same filename as before.
- Updated `fleetctl query --hosts` to work with hostnames, host UUIDs, and/or hardware serial numbers.
- Added support to `fleetctl debug connection` to test TLS connection with the embedded certs.pem in
  the fleetctl executable.
- Added host's display name to calendar event descriptions.
- Added .yml and .yaml file type validation and error message to `fleetctl apply`.

### Device Management (MDM)

- Added iOS/iPadOS builtin manual labels. 
  - **NOTE:** Before migrating to this version, make sure to delete any labels with name "iOS" or "iPadOS".
- Added aggregation of iOS/iPadOS OS versions.
- Added change to custom profiles for iOS/iPadOS to go from 'pending' straight to 'verified' (skip 'verifying').
- Added support for renewing SCEP certificates with custom enrollment profiles.
- Added automatic install of `fleetd` when a host turns on MDM now uses the latest released `fleetd` version.
- Added support for `END_USER_EMAIL` and `FLEET_DESKTOP` parameters to Windows MSI install package.
- Added API changes to support the `labels_include_all` and `labels_exclude_any` fields (and accept the deprecated `labels` field as an alias for `labels_include_all`).
- Added `fleetctl gitops` and `fleetctl apply` support for `labels_include_all` and `labels_exclude_any` to configure a custom setting.
- Added UI for uploading custom profiles with a target of hosts that include all/exclude any selected labels.
- Added the database migrations to create the new `exclude` column for labels associated with MDM profiles (and declarations).
- Updated the profile reconciliation logic to handle the new "exclude any" labels.
- Updated so that the `fleetd` cleanup script for macOS that will return completed when run from Fleet.
- Updated so that the `fleetd` uninstall script will return completed when run from Fleet.
- Updated script run permissions -- only admins and maintainers can run arbitrary or saved scripts (not observer or observer+).
- Updated `fleetctl get mdm_commands` to return 20 rows and support `--host` `--type` filters to improve response time.
- Updated the instructions for manual MDM enrollment on the "My device" page to be clearer and align
  with Apple updates.
- Disabled credential caching and reboot on Windows lock.

### Vulnerability Management

- Added "Vulnerable" filter to the host details software table.

### Bug fixes and improvements

- Updated Go version to go1.22.4.
- Updated to render only one banner on the my device page based on priority order.
- Updated software updated timestamp tooltip.
- Removed DB error message from the UI when showing a error response.
- Updated fleetctl get queries/labels/hosts descriptions.
- Reinstated ability to sort policies by passing count.
- Improved the accuracy of the heuristic used to deterimine if a host is connected to Fleet via MDM by using osquery data for hosts that didn't send a Checkout message.
- Improved the matching of `pkg` installer files to existing software.
- Improved extraction of application name from `pkg` installers.
- Clarified various help and error texts around host identifiers.
- Hid CTA on inherited queries/policies from team level users.
- Hid query delete checkboxes from team observers.
- Hid "Self-service" in Fleet Desktop and My device page if there is no self-service software available.
- Hid the host detail page's "Run script" action from Global and Team Observer/+s.
- Aligned the "View all hosts" links in the Software titles and versions tables.
- Fixed Microsoft Office June 2024 false negative vulnerabilities and added custom vulnerability matching.
- Fixed issue where some Windows applications were getting matched against Windows OS vulnerabilities.
- Fixed counts for hosts with with low disk space in summary page.
- Fixed allowing Observer and Observer+ roles to download software installers.
- Fixed crash in `fleetd` installer on Windows if there are registry keys with special characters on the system.
- Fixed `fleetctl debug connection` to support server TLS certificates with intermediates.
- Fixed macOS declarations being stuck in "to be removed" state indefinitely.
- Fixed link to `fleetd` uninstall instructions in "Delete device" modal.
- Fixed exporting CSVs with fields that contain commas to render properly.
- Fixed issue where the Fleet UI could not be used to renew the ABM token after the ABM user who created the token was deleted.
- Fixed styling issues with the target inputs loading spinner on the run live query/policy page.
- Fixed an issue where special characters in HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall breaks the "installer_utils.ps1 -uninstallOrbit" step in the Windows MSI installer.
- Fixed various UI capitalizations.

## Fleet 4.53.1 (Jul 01, 2024)

### Bug fixes

* Updated fleetctl get queries/labels/hosts descriptions.
* Fixed exporting CSVs with fields that contain commas to render properly.
* Fixed link to fleetd uninstall instructions in "Delete device" modal.
* Rendered only one banner on the my device page based on priority order.
* Hidden query delete checkboxes from team observers.
* Fixed issue where the Fleet UI could not be used to renew the ABM token after the ABM user who created the token was deleted.
* Fixed an issue where special characters in HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall broke the "installer_utils.ps1 -uninstallOrbit" step in the Windows MSI installer.
* Fixed counts for hosts with low disk space in summary page.
* Fleet UI fixes: Hide CTA on inherited queries/policies from team level users.
* Updated software updated timestamp tooltip.
* Fixed issue where some Windows applications were getting matched against Windows OS vulnerabilities.
* Fixed crash in `fleetd` installer on Windows if there are registry keys with special characters on the system.
* Fixed UI capitalizations.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.54.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-07-17">
<meta name="articleTitle" value="Fleet 4.54.0 | Maintenance window improvements, target hosts via label exclusion.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.54.0-1600x900@2x.png">
