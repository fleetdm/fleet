# Fleet 4.67.0 | Foreign vitals, policy targets, cancel activities

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/5L6Z7Vo2mBk?si=OfkVVcBdHfCeAk5y" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.67.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.67.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Foreign vitals
- Policy targets
- Cancel activities

### Foreign vitals

Fleet now pulls end user details from your identity provider (IdP)—like IdP email, full name, and group memberships—into host vitals. This makes it easier to identify who is using each host to speed up troubleshooting and audits. Learn more [here](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).

### Policy targets

Security engineers can now target policies to specific hosts using labels. This gives teams more precise control over enforcement within a [Fleet team](https://fleetdm.com/guides/teams), helping apply the right checks to the right hosts.

### Cancel activities

IT admins can now cancel activities—like software installs or scripts—before they run. This helps correct mistakes or reprioritize tasks without waiting for actions to complete.

## Changes

### Security Engineers
- Added ability to set labels on policies via GitOps.
- Added backend support for labels on policies.
- Added ability to cancel upcoming host activities in the UI.
- Added the `DELETE /api/latest/fleet/hosts/:id/activities/upcoming/:activity_id` endpoint to cancel an upcoming activity for a host.
- Added support for native Windows ARM64 in fleetd (`fleetctl package --arch=arm64 --type=msi`).

### IT Admins
- Added SCIM integration, which allows IdP email, full name, and groups to be visible in host vitals. SCIM data is also used for getting the end user's full name during end user authentication of macOS setup flow, if needed. Currently, only Okta IdP is supported.
- Added a new IDP section to the integrations page where users can see their SCIM connection status.
- Added new users card on host details and my device page that shows host end user and IDP information.
- Added ability to set labels on policies via GitOps.
- Added backend support for labels on policies.
- Added ability to cancel upcoming host activities in the UI.
- Added the `DELETE /api/latest/fleet/hosts/:id/activities/upcoming/:activity_id` endpoint to cancel an upcoming activity for a host.
- Added support for native Windows ARM64 in fleetd (`fleetctl package --arch=arm64 --type=msi`).
- Added logging for invalid Windows MDM SOAP message and return 400 instead of 5XX to help debug Windows MDM issues.
- Removed Apple MDM profile validation checks for com.apple.MCX keys (dontAllowFDEDisable and dontAllowFDEEnable) due to customer feedback.
- Fixed a bug where BYOD iDevices deleted in Fleet but still enrolled in MDM were not re-created on the next MDM checkin.
- Fixed an issue with how names for macOS software titles were calculated and prevents duplicate entries being created if the software is renamed by end users.

### Other improvements and bug fixes
- Added support for `vmodule` hidden osquery flag to assist with debugging.
- Added an additional statistic item to count ABM pending hosts.
- Added a timeout so the desktop app retries if not displayed after 1 minute.
- Updated UI to allow adding labels when saving or editing polices.
- Included newly created host ids in activities generated when hosts enroll in fleet.
- Moved view all host link onto host count of software, OS, and vulnerability details pages
- Updated Go to v1.24.1.
- Updated UI tables to truncate with tooltips for software, query, and policy names and improved keyboard accessibility to those clickable elements.
- Updated to accept any "http://" or "https://" prefixed URL to allow for easier testing.
- Updated apmhttp package to fix upload of medium/big sized software packages in environments where APM tracing is enabled.
- Fixed UI Gitops Mode getting cleared when other settings are modified.
- Fixed invalid default serial numbers being displayed for some hosts.
- Fixed pagination resetting the platform filter on the operating system UI table.
- Fixed issue where `fleetctl gitops --dry-run` would sometimes fail when creating and using labels in the same run.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.67.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-04-24">
<meta name="articleTitle" value="Fleet 4.67.0 | Foreign vitals, policy targets, cancel activities">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.67.0-1600x900@2x.png">