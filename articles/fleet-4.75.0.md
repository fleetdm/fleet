# Fleet 4.75.0 | Omarchy Linux, Android configuration profiles, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/a6qx4th3dKs?si=KYVIvqZTb9AZM27Y" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.75.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.75.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

> Fleet added Santa tables: `santa_allowed`, `santa_denied`, `santa_status`. If you already deploy a custom Santa extension (like [Trail of Bits](https://github.com/trailofbits/osquery-extensions/tree/master/santa)) with tables that have the same names (exactly), Fleet's agent will [crash](https://github.com/fleetdm/fleet/issues/34789). To resolve, update variables in [this script](https://github.com/fleetdm/fleet/tree/11984cdf6fad6797e0be7d1ce927d6d9c19d51c0/docs/solutions/macOS/scripts/uninstall-santa-extension.sh) and run it on macOS hosts to uninstall your custom Santa extension.

## Highlights

- Arch Linux / Omarchy Linux
- Android configuration profiles
- Smallstep certificates
- Fleet UI refresh
- Labels page
- Easy-to-read MDM commands

### Arch Linux / Omarchy Linux

Fleet now supports [Arch Linux](https://archlinux.org/) and [Omarchy](https://omarchy.org/) Linux. You can view host vitals like software inventory, run scripts, and install software.

### Android configuration profiles

You can now apply custom settings to work profiles on employee-owned (BYOD) Android hosts using configuration profiles. This lets you keep Android hosts compliant and secure. Learn how to create in [this video](https://www.youtube.com/watch?v=Jk4Zcb2sR1w).

### Smallstep certificates

Fleet now integrates with [Smallstep](https://smallstep.com/) as a certificate authority. You can deliver Wi-Fi/VPN [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificates to macOS, iOS, and iPadOS hosts to automate secure network access for your end users. Learn more in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#smallstep).

### Labels page

A new **Labels** page makes it easier to view and edit labels. You can find the new **Labels** page in Fleet by selecting your avatar in the top-right corner and selecting **Labels**.

### Fleet UI refresh

Fleet’s UI just got a refresh. Consistent spacing, colors, and layouts now span every page, from fleetdm.com to the Fleet product. Expect a more polished, accessible experience whether you're managing hosts or running live queries.

### Easy-to-read MDM commands

Long MDM payloads and outputs are now easier to read and copy, thanks to a new vertical layout in the `fleetctl get mdm command` results. Learn more about fleetctl in [the guide](https://fleetdm.com/guides/fleetctl).

## Changes

### Security Engineers
- Added support for Smallstep certificate authority.
- Added false-positive filtering for Linux vulnerability scanning.
- Added support for Arch Linux hosts.
- Added software inventory ingestion from Arch Linux hosts.
- Added new rate limiting implementation for Fleet Desktop API endpoints to support all/many hosts of a deployment behind NAT (single IP).
- Added support for reading server `private_key` from AWS Secrets Manager.
- Added support for vulnerabilities feed CPE translation JSON to override `sw_edition` field.
- Added filter for removing duplicate RPM python packages and renaming pip packages to match OVAL definitions (same as Ubuntu).
- Added ability to specify a Fleet host ID when declaring a manual label in a Gitops YAML file.
- Added a dedicated page, table, and logical integrations with other parts of the UI for managing labels.

### IT Admins
- Added configuration profile support for Android hosts.
- Added activity logging for Android profile creation, modification, and deletion.
- Added support for software installation during Windows setup experience.
- Added support for Arch Linux hosts.
- Added software inventory ingestion from Arch Linux hosts.
- Added support to `fleetctl` to generate `fleetd` installers for Arch Linux (`.pkg.tar.zst`).
- Added software name into checksum calculation for macOS apps.
- Added ability to specify a Fleet host ID when declaring a manual label in a Gitops YAML file.
- Added a dedicated page, table, and logical integrations with other parts of the UI for managing labels.
- Added OpenTelemetry instrumentation to scheduled jobs and several API endpoints.
- Added CRON job to reconcile Android profiles.
- Added retries with backoff when Apple's assets API fails with a timeout error.
- Added ability to unenroll personal iOS/iPadOS devices from Fleet.
- Added support for assigning host labels based on idP attributes for iOS and iPadOS hosts.
- Added ability to turn off MDM for iOS and iPadOS devices when refetcher returns device token is inactive.
  > Note: The package will need to be updated out-of-band once, because the pre-removal script from previously-generated packages is called upon an upgrade. The old pre-removal script stopped Orbit unconditionally.
- Added support for hosts enrolled with Company Portal using the legacy SSO extension (for Entra's conditional access).

### Other improvements and bug fixes
- Updated DEB and RPM packages generated by `fleetctl package` to now be safe to upgrade in-band through the Software page.
- Updated to return count in list host certificates API response, and use it in the certificate table.
- Updated setup experience to try software installs up to 3 times by default in case of intermittent failures.
- Modified the Apple profile reconciliation CRON logic to query for installs and removals within a transaction to avoid race conditions around team or label changes.
- Fixed inconsistent spacing in Controls OS settings headers.
- Validated setting `manual_agent_install` option on the server.
- Ignore warning when LastOpenedAt for software is nil on macOS.
- Improved install action tooltips and modals including timestamps to VPP successful installs.
- Changed the response code for UserAuthenticate checkin messages, which are unsupported, from a 5XX to "410 Gone" as specified in the Apple MDM protocol docs for servers that do not implement this method.
- Ensured UI consistency by adding a border to the empty state of End User Authentication section.
- Added easy to understand error messages when configuring Entra conditional access in Fleet.
- Updated docs for the `pwd_policy` table to better reflect the meaning of `days_to_expiration`.
- Improved the layout of the IdP-driven label form.
- Updated Hosts table > hostname column to truncate overflowing hostnames and place the full name in a tooltip on hover.
- Removed duplicate tar.gz copies of osqueryd and Fleet Desktop from built packages (DEB/RPM/PKG).
- Extended the number of errors Fleet looks for when determining whether we should invalidate the prepared statements cache.
- Updated instructions in Linux key escrow modal.
- Adjusted log level to "info" instead of "error" when Windows MDM endpoints generate client errors (e.g. empty binary security token).
- Disabled debug logging by default in `fleetctl preview` and reformatted login information.
- Improved handling of host details page label pills for labels with very long names.
- Modified Controls > OS settings > Custom settings so profile upload time is based on `updated_at` instead of `created_at`. 
- Added check to GitOps command to throw error if positional arguments are detected.
- Added an error message when software is defined in a package YAML file in GitOps but some fields expected in that file were set at the team level. Previously, GitOps would silently ignore the fields set at the team level in this case.
- Updated the OS updates current versions empty state to match consistancy with other empty states.
- Updated message shown in the 'Delete Script' modal.
- Added a delay to the platform compatibility tooltip showing when creating or editing a query.
- Added error when uploading signed profiles instead of when trying to deliver them.
- Updated old end user migration workflow preview, and switch to video for product consistency.
- Replaced outdated Firefox icon with a new one that follows brand guidelines.
- Updated UI to make policy pass/fail icons and copy consistent across host details, my device, and manage policies tables.
- Removed the software renaming fix introduced in 4.73.3 due to MySQL DB performance issues.
- Optimized software ingestione rename functionality to generate less lock contention during high concurrency.
- Optimized ingestion of software names on macOS apps when vendor-supplied bundle executable names are unclear.
- Optimized software title reconciliation in vulnerabilities cron job.
- Revised macOS software ingestion to correctly show application names for Steam games instead of `run.sh`.
- Added logic to detect and fix migration issues caused by improperly published Fleet v4.73.2 Linux binary.
- Updated go to 1.25.1.
- Fixed inconsistent subtitle text style in Custom Settings.
- Fixed SentinelOne pkg generating wrong bundle identifier for auto-install policy.
- Fixed required query parameters using field name instead of parameter name in error messages
- Fixed a bug where blocking of VPP installs on personally enrolled Apple devices was not in place.
- Fixed edit teams action in VPP table dropdown not being blocked when Fleet is in GitOps mode.
- Fixed certificate ingest parser to no longer break on multiple equal signs in certificate key pair values.
- Fixed certificate ingest parser to allow for only multiple relative distinguished names separated by `+`.
- Fixed 422 error when hitting `/api/v1/fleet/commands` endpoint with team filter.
- Fixed deletion of conditional access integration by adding a spinner and clearing the tenant ID after the deletion.
- Fixed an issue on ChromeOS and Windows where the cursor in the SQL editor is misaligned.
- Fixed issue where "Controls" link in the top nav didn't always go to the default controls page.
- Fixed cases where Firefox ESR installations would have false-positive vulnerabilities reported that were backported to the ESR.
- Fixed clicking the currently selected navbar item would cause a full-page rerender.
- Fixed EULA path to be relative to the YAML file in `fleetctl gitops`, as it is for other settings.
- Fixed bundle identifier for privileges macos software pkg and fixed existing software installers to use corrected software title. The privileges application should show the correct status in software inventory.
- Fixed the reported version of fleetd on the Software tab for Linux hosts.
- Fixed invalid GET and DELETE requests that incorrectly included request bodies in client code, ensuring HTTP compliance.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.75.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-10-17">
<meta name="articleTitle" value="Fleet 4.75.0 | Omarchy Linux, Android configuration profiles, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.75.0-1600x900@2x.png">
