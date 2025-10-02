# Fleet 4.72.0 | Account-based user enrollment, smarter self-service, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/-N1eZ-nw59A?si=QYbQtTBazOjHR0PG" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.72.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.72.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Account-based user enrollment for iOS/iPadOS
- Smarter self-service
- More Fleet-maintained apps
- Linux host identity certificates

### Account-based user enrollment for iOS/iPadOS

Users can now enroll personal iPhones and iPads directly via the **Settings** app by signing in with a Manager Apple Account(work email). This makes it easy to apply only the necessary controls for accessing org tools—without compromising personal privacy. Learn more in [the guide](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account).

### Smarter self-service

Fleet Desktop now shows only the relevant software actions (install, update, uninstall) based on the actual state of the app on each machine. End users see exactly what they can do—nothing more, nothing less. Learn more about self-service software in [the guide](https://fleetdm.com/guides/software-self-service).

### More Fleet-maintained apps

You can now manage these popular apps as Fleet-maintained software—no need to hunt down vendor installers or build packages yourself. Just select and deploy. Learn more about [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps).

### Linux host identity certificates

Fleet now supports TPM-based identity for Linux hosts. When you deploy [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd), it can automatically obtain a hardware-backed identity certificate (similar to macOS MDM). This certificate is required to communicate with the Fleet server—enhancing trust and tamper resistance for your Linux fleet.

## Changes

### Security Engineers
- Added support for issuing host identity certificates through SCEP (Simple Certificate Enrollment Protocol) that `fleetd` can use with TPM 2.0 hardware to cryptographically sign all HTTP requests.
- Added flag `--fleet-managed-host-identity-certificate` to generate `fleetd` packages for linux that use TPMs to sign HTTP requests.
- Added `sso_server_url` configuration option to support SSO setups with separate URLs for admin access vs agent/API access. When set, SSO authentication will only work from the specified URL. This fixes SSO authentication errors for organizations using dual URL configurations.

### IT Admins
- Added support for Apple Account Driven User Enrollment for iOS/iPadOS when end user authentication is configured.
- Added support for MS-MDE2 v7.0 Windows MDM Enrollments.
- Added the following Fleet-maintained apps for macOS: iTerm2, Yubikey Manager, VNC Viewer, Beyond Compare. 
- On the host details > software > library page and Fleet Desktop > Self-service page, show installer status and installer actions based on what software is detected in software inventory.
- On the host details > software > library page and Fleet Desktop > Self-service page, show user's when a software can be updated, allowing users to easily trigger a software update and see fresh data after an update completes.
- Updated VPP apps reported by osquery to retain their last install information when viewed in host software library.
- Switched to more comprehensive `UpgradeCode` based uninstall scripts when an `UpgradeCode` can be extracted from an MSI custom package.

### Other improvements and bug fixes
- Added support for `fleetd` TUF extensions on Linux arm64 and Windows arm64 devices.
- Added a fallback to package install path for extracting app names from uploaded PKG packages.
- Added special handling for version extraction of Fleet-maintained app manifests that reference a download URL that isn't version-pinned.
- Improved `fleetctl gitops` type error mesages.
- Improved accuracy of auto-install queries for custom MSI packages by using a better identifier.
- Label created_at no longer factored in when scoping software packages by "exclude any" manual labels.
- Refactored `AddHostsToTeam` method to fix race condition introduced by global var.
- Changed `enable_software_inventory` to default to true if missing from gitops config.
- Modified backend for `GET /api/v1/fleet/commands` when filtering by `host_identifier` to address performance concerns and exhausting database connections when API is called concurrently for many hosts.
- Allowed users of Fleet in Primo mode to access Software automations and failing policy ticket & webhook automations.
- Update UI to support personally enrolled MDM devices.
- Removed DEB and RPM installers from installable software lists on hosts with incompatible Linux distributions (e.g. Ubuntu for an RPM).
- Revised MSI uninstall scripts to wait for an uninstall to complete before returning and avoid restarting after an uninstall.
- Added back software mutation on ingestion to fix non-semver-compliant software versions, starting with DCV Viewer.
- Increased timeouts on `/fleet/mdm/profiles/batch` to better support customer workflows with large numbers of profiles.
- Made consistent and update the Install and Uninstall detail modals for VPP and non-VPP apps across the Fleet UI.
- Updated go to 1.24.6.
- Fixed issue with package ids ordering causing software installers' scripts to be inconsistently generated.
- Fixed incorrectly displayed status in controls OS Settings page, if a host was only pending or failing on declaration for removal.
- Fixed bug with `mdm_bridge` Orbit table that caused panics due to invalid COM initialization.
- Fixed bug where a certificate Distinguished Name (DN) parser did not allow forward slashes in the value which resulted in parsing error.
- Fixed an issue where the detected date for software vulnerabilities was not being pulled correctly from the database.
- Fixed missing empty host lists on manual labels in gitops.
- Fixed an issue where two banners would sometimes be displayed on the host details page.
- Fixed missing webhook url in automations tooltip.
- Fixed an issue where using `ESCAPE` in a `LIKE` clause caused SQL validation to fail.
- Fixed error when trying to escrow a linux disk key multiple times.
- Fixed silent failure when passing flags after arguments in `fleetctl`.
- Fixed wrongly formatted URL for EULA when accessing from Fleet UI and when shown in the iFrame for SSO callback.
- Fixed stale pending remove apple declarations, if the host was offline while adding and removing the same declaration.
- Fixed a case where a vulnerability would show up twice for a given operating system.
- Fixed specification of policy software automations via GitOps when referring to software by hash from a software YAML file.
- Fixed cases where the vulnerabilities list endpoint would count the same CVE multiple times for the `count` field returned with a result set.
- Fixed an issue where SSO URLs with trailing slashes would cause authentication failures due to double slashes in the ACS URL. Both regular SSO and MDM SSO URLs now properly handle trailing slashes.
- Fixed an issue during the DEP sync where errors such as 404 from the DEP API could result in devices never being assigned a cloud configuration profile.
- Fixed server panic when listing software titles for "All teams" with page that contains a software title with a policy automation in "No team".

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.72.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-08-13">
<meta name="articleTitle" value="Fleet 4.72.0 | Account-based user enrollment, smarter self-service, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.72.0-1600x900@2x.png">
