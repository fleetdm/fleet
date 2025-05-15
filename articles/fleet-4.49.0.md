# Fleet 4.49.0 | VulnCheck's NVD++, device health API, `fleetd` data parsing.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/UQEQZV_puHg?si=J6BE0ch56CSDMP5d" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.49.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.49.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Enhancing Fleet's vulnerability management with VulnCheck integration
* Device health API includes critical policy and resolution data
* `fleetd` data parsing expansion
* Apply labels using UI or API
* Resend configuration profiles



### Enhancing Fleet's vulnerability management with VulnCheck integration

Fleet is integrating VulnCheck to enhance our vulnerability management capabilities, ensuring our users can manage Common Platform Enumeration (CPE) data more effectively and securely. Utilizing VulnCheck's NVD++ service, Fleet will provide reliable, timely access to vulnerability data, overcoming delays and inconsistencies in the National Vulnerability Database (NVD). This integration improves the accuracy and timeliness of threat detection and streamlines the overall vulnerability management process, empowering IT administrators to identify and mitigate security threats swiftly. Learn more about how this enhancement strengthens Fleet's security framework in our latest blog post: [Enhancing Fleet's Vulnerability Management with VulnCheck Integration](https://fleetdm.com/announcements/enhancing-fleets-vulnerability-management-with-vulncheck-integration).


### Device health API includes critical policy and resolution data

Fleet has updated its device health API to include critical and policy resolution data, enhancing the utility of this API for specific workflow conditions where compliance verification is essential before proceeding. This update allows for real-time authentication checks to ensure a host complies with set policies, thereby supporting secure and compliant operational workflows. By integrating critical compliance data into the device health API, Fleet enables administrators to enforce and verify security policies efficiently, ensuring that only compliant devices proceed in sensitive or critical operations. This enhancement supports thorough compliance management and reinforces secure practices within IT environments, streamlining processes where policy adherence is crucial.


### `fleetd` data parsing expansion

Fleet's agent (`fleetd`) has expanded its data parsing capabilities by adding support for JSON, JSONL, XML, and INI file formats as tables. This functionality allows for more versatile data extraction and management, enabling users to convert these popular data formats directly into queryable tables. This capability is particularly useful for IT and security teams who need to analyze and monitor configuration and data files across various systems within their digital environments efficiently. By facilitating integration and manipulation of data from these diverse formats, Fleet helps ensure that teams can maintain better oversight and faster responsiveness when managing operational and security needs. This feature is a natural extension of Fleet's ongoing efforts to empower IT professionals with comprehensive tools for robust data handling and security management.


### Apply labels using UI or API

Fleet has expanded the flexibility of label management by enabling users to add labels manually through both the UI and API. This capability was previously available only via the CLI. This enhancement allows administrators to more conveniently categorize and manage hosts directly within the user interface or programmatically via the API, aligning with various operational workflows. By streamlining the label application process, Fleet makes it easier for teams to organize and access host data according to specific criteria, thereby improving operational efficiency and responsiveness. This update supports better integration and automation capabilities within IT environments, empowering users to maintain organized and effective device management practices.


### Resend configuration profiles

Fleet has introduced a new feature that allows users to resend a configuration profile to a host, which is crucial for maintaining current settings and certificates. This functionality is particularly beneficial in scenarios where renewing SCEP certificates, signing certificates need updating, or reapplication of existing configurations is required to ensure continuity and compliance. By enabling the reissuance of configuration profiles directly from the platform, Fleet supports continuous device management and security upkeep, facilitating a proactive approach to maintaining and securing digital environments. This feature enhances Fleet's utility for administrators by simplifying the management of device configurations.



## Changes

### Endpoint operations

- Added integration with Google Calendar for policy compliance events.
- Added new API endpoints to add/remove manual labels to/from a host.
- Updated the `POST /api/v1/fleet/labels` and `PATCH /api/v1/fleet/labels/{id}` endpoints to support creation and update of manual labels.
- Implemented changes in `fleetctl gitops` for batch processing queries and policies.
- Enabled setting host status webhook at the team level via REST API and fleetctl apply/gitops.

### Device management (MDM)

- Added API functionality for creating DDM declarations, both individually and as a batch.
- Added creation or update of macOS DDM profile to enforce OS Updates settings whenever the settings are changed.
- Updated `fleetctl run-script` to include new `--team` and `--script-name` flags.
- Displayed disk encryption status in macOS as "verifying" while verifying the escrowed key.
- Added the `enable_release_device_manually` configuration setting for teams and no team, which controls the automatic release of a macOS DEP-enrolled device.
- Updated the `POST /api/v1/fleet/hosts/:id/wipe` Fleet Premium API endpoint to support remote wiping a host.
- Added the `enable_release_device_manually` configuration, which affects macOS automatic enrollment profile settings.

### Vulnerability management

- Ignored Valve Corporation's Steam client's vulnerabilities on Windows and macOS due to retrieval challenges of the true version.
- Updated the GET fleet/os_versions and GET fleet/os_versions/[id] to restrict team users from accessing os versions on hosts from other teams.

### Bug fixes and improvements

- Upgraded Golang version to 1.21.7.
- Added a minimum supported node version in the `package.json`.
- Made block_id mismatch errors more informative as 400s instead of 500s.
- Added Windows MDM support to the `osquery-perf` host-simulation command.
- Updated calendar events automations to not show error validation on enabling the feature.
- Migrated MDM-related endpoints to new paths while maintaining support for old endpoints indefinitely.
- Added a missing database index to the MDM Windows enrollments table to improve performance at scale.
- Added cross-platform check for duplicate MDM profiles names in batch set MDM profiles API.
- Fixed a bug where Microsoft Edge was not reporting vulnerabilities.
- Fixed an issue with the `20240327115617_CreateTableNanoDDMRequests` database migration.
- Fixed the error message to indicate if a conflict on uploading an Apple profile was caused by the profile's name or its identifier.
- Fixed license checks to allow migration and restoring DEP devices during trial.
- Fixed a 500 error in MySQL 8 and when DB user has insufficient privileges for `fleetctl debug db-locks` and `fleetctl debug db-innodb-status`.
- Fixed a bug where values not derived from "actual" fleetd-chrome tables were not being displayed correctly.
- Fixed a bug where values were not being rendered in host-specific query reports.
- Fixed an issue with automatic release of the device after setup when a DDM profile is pending.
- Fixed UI issues: alignment bugs, padding around empty states, tooltip rendering, and incorrect rendering of the global Host status expiry settings page.
- Fixed a bug where `null` or excluded `smtp_settings` caused a UI 500 error.
- Fixed an issue where a bad request response from a 3rd party MDM solution would result in a 500 error in Fleet during MDM migration.
- Fixed a bug where updating policy name could result in multiple policies with the same name in a team.
- Fixed potential server panic when events are created with calendar integration, but then global calendar integration is disabled.
- Fixed fleetctl gitops dry-run validation issues when enabling calendar integration for the first time.
- Fixed a bug where all Windows MDM enrollments were detected as automatic.

## Fleet 4.48.3 (Apr 16, 2024)

### Bug fixes

* Updated calendar webhook to retry if it receives response 429 "Too Many Requests". Webhook request will retry for 30 minutes with a 1 minute max delay between retries.
* Updated label endpoints and UI to prevent creating, updating, or deleting built-in labels.
* Fixed edge cases of team ID being lost in various flows.
* Fixed queries to correctly parse params for `GET` ...`policies/count`, `GET` ...`teams/:id/policies/count`, and `GET` ...`vulnerabilities`.
* Fixed 'GET` ...`labels` to return `400` when the non-supported `query` url param was included in the request. Previous behavior was to silently ignore that param and return `200`.
* Casted windows exit codes to signed integers to match windows interpreter.
* Fixed a bug where some scripts got stuck in "upcoming" activity permanently.
* Fixed a bug where the translate API returned "forbidden" instead of "bad request" for an empty JSON body.
* Fixed an uncaught bug where "forbidden" would be returned for invalid payload type, which should also be a bad request.
* Fixed an issue where applying Windows MDM profiles using `fleetctl apply` would cause Fleet to overwrite the reserved profile used to manage Windows OS updates.
* Fixed a bug where we were not ignoreing leading and trailing whitespace when filtering Fleet entities by name.
* Fixed a bug where query retrieving bitlocker info from windows server wouldn't return.
* Fixed MDM migration starting when the device didn't have the right ADE JSON profile already assigned.

## Fleet 4.48.2 (Apr 09, 2024)

### Bug fixes

* Fixed an issue with the `20240327115617_CreateTableNanoDDMRequests` database migration where it could fail if the database did not default to the `utf8mb4_unicode_ci` collation.
* Fixed an issue with automatic release of the device after setup when a DDM profile is pending.

## Fleet 4.48.1 (Apr 08, 2024)

### Bug fixes

- Made block_id mismatch errors more informative as 400s instead of 500s
- Fixed a bug where values were not being rendered in host-specific query reports
- Fixed potential server panic when events are created with calendar integration, but then global calendar integration is disabled




## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.49.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-04-23">
<meta name="articleTitle" value="Fleet 4.49.0 | VulnCheck's NVD++, device health API, fleetd data parsing.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.49.0-1600x900@2x.png">
