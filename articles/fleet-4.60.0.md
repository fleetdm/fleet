# Fleet 4.60.0 | Escrow Linux disk encryption keys, custom targets for OS settings, scripts preview

![Fleet 4.60.0](../website/assets/images/articles/fleet-4.60.0-1600x900@2x.png)

Fleet 4.60.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.60.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Escrow Linux disk encryption keys
- Custom targets for OS settings
- Preview scripts before run

### Escrow Linux disk encryption keys

Fleet now supports escrowing the disk encryption keys for Linux (Ubuntu and Fedora) workstations. This means teams can access encrypted data without needing the local password when an employee leaves, simplifying handoffs and ensuring critical data remains accessible while protected. Learn more in the guide [here](https://fleetdm.com/guides/enforce-disk-encryption).

### Custom targets for OS settings

With Fleet, you can now use a new "include any" label option to target OS settings (configuration profiles) to specific hosts within a team. This added flexibility allows for finer control over which OS settings apply to which hosts, making it easier to tweak configurations without disrupting broader baselines (Fleet [teams](https://fleetdm.com/guides/teams)).

### Preview scripts before run

Fleet now provides the ability to preview scripts directly on the **Host details** or **Scripts** page. This quick-view feature reduces the risk of errors by letting you verify the script is correct before running it, saving time and ensuring smoother operations.

## Changes

### Endpoint operations
- Added support for labels_include_any to gitops.
- Added major improvements to keyboard accessibility throughout app (e.g. checkboxes, dropdowns, table navigation).
- Added activity item for `fleetd` enrollment with host serial and display name.
- Added capability for Fleet to serve YARA rules to agents over HTTPS authenticated via node key (requires osquery 5.14+).
- Added a query to allow users to turn on/off automations while being transparent of the current log destination.
- Updated UI to allow users to view scripts (from both the scripts page and host details page) without downloading them.
- Updated activity feed to generate an activity when activity automations are enabled, edited, or disabled.
- Cancelled pending script executions when a script is edited or deleted.

### Device management (MDM)
- Added better handling of timeout and insufficient permissions errors in NDES SCEP proxy.
- Added info banner for cloud customers to help with their windows autoenrollment setup.
- Added DB support for "include any" label profile deployment.
- Added support for "include any" label/profile relationships to the profile reconciliation machinery.
- Added `team_identifier` signature information to Apple macOS applications to the `/api/latest/fleet/hosts/:id/software` API endpoint.
- Added indicator of how fresh a software title's host and version counts are on the title's details page.
- Added UI for allowing users to install custom profiles on hosts that include any of the defined labels.
- Added UI features supporting disk encryption for Ubuntu and Fedora Linux.
- Added support for deb packages compressed with zstd.

### Vulnerability management
- Allowed skipping computationally heavy population of vulnerability details when populating host software on hosts list endpoint (`GET /api/latest/fleet/hosts`) when using Fleet Premium (`populate_software=without_vulnerability_descriptions`).

### Bug fixes and improvements
- Improved memory usage of the Fleet server when uploading a large software installer file. Note that the installer will now use (temporary) disk space and sufficient storage space is required.
- Improved performance of adding and removing profiles to large teams by an order of magnitude.
- Disabled accessibility via keyboard for forms that are disabled via a slider.
- Updated software batch endpoint status code from 200 (OK) to 202 (Accepted).
- Updated a package used for testing (msw) to improve security.
- Updated to reboot linux machine on unlock to work around GDM bug on Ubuntu 24.04.
- Updated GitOps to return an error if the deprecated `apple_bm_default_team` key is used and there are more than 1 ABM tokens in Fleet.
- Dismissed error flash on the my device page when navigating to another URL.
- Modified the Fleet setup experience feature to not run if there is no software or script configured for the setup experience.
- Set a more accurate minimum height for the Add hosts > ChromeOS > Policy for extension field, avoiding a scrollbar.
- Added UI prompt for user to reenter the password if SCEP/NDES url or username has changed.
- Updated ABM public key to download as as PEM format instead of CRT.
- Fixed issue with uploading macOS software packages that do not have a top level `Distribution.xml`, but do have a top level `PackageInfo.xml`. For example, Okta Verify.app.
- Fixed some cases where Fleet Maintained Apps generated incorrect uninstall scripts.
- Fixed a bug where a device that was removed from ABM and then added back wouldn't properly re-enroll in Fleet MDM.
- Fixed name/version parsing issue with PE (EXE) installer self-extracting archives such as Opera.
- Fixed a bug where the create and update label endpoints could return outdated information in a deployment using a mysql replica.
- Fixed the MDM configuration profiles deployment when based on excluded labels.
- Fixed gitops path resolution for installer queries and scripts to always be relative to where the query file or script is referenced. This change breaks existing YAML files that had to account for previous inconsistent behavior (e.g. installers in a subdirectory referencing scripts elsewhere).
- Fixed issue where minimum OS version enforcement was not being applied during Apple ADE if MDM IdP integration was enabled.
- Fixed a bug where users would be allowed to attempt an install of an App Store app on a host that was not MDM enrolled.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.60.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2024-11-25">
<meta name="articleTitle" value="Fleet 4.60.0 | Escrow Linux disk encryption keys, custom targets for OS settings, scripts preview">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.60.0-1600x900@2x.png">