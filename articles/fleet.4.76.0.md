# Fleet 4.76.0 | Self-service scripts, JetBrains/Cursor/Windsurf vulnerabilities, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/2hJ7yZTBaVY?si=11HG8r-mS1iF9fma" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.76.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.76.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Self-service scripts
- Vulnerabilities for Cursor, Windsurf, and JetBrains extensions
- Improved macOS, iOS, and iPadOS setup experience
- Android software inventory
- Lock (Lost Mode) for iOS and iPadOS
- New Fleet-maintained apps

### Self-service scripts

You can now create custom Linux and Windows packages that include just a script (aka payload-free packages). In Fleet, head to **Software** page and select **Add software > Custom package**. This is perfect for self-service utilities or bundling multiple scripts as part of your out-of-the-box setup experience. macOS self-service scripts are coming soon.

### Vulnerabilities for JetBrains, Cursor, and Windsurf extensions

Vulnerabilities (CVEs) in all Cursor, Windsurf, other VSCode forks, and JetBrains IDE extensions now show up in the **Software**, **Host details**, and **My device** pages. Gain better coverage of high-risk developer tools. Learn more about CVEs in the [vulnerabilities guide](https://fleetdm.com/guides/vulnerability-processing#basic-article).

### Improved macOS, iOS, and iPadOS setup experience

During out-of-the-box macOS setup, if critical software fails to install during setup, Fleet now cancels the process and shows an error. This ensures end users run through setup again and, if they're still running into issues, contact IT before moving forward. This helps avoid misconfigured hosts in production.

For iOS and iPadOS, installing apps on company-owned iPhones and iPads during enrollment is now supported. Perfect for instantly setting up kiosk devices, shared iPads, or Zoom rooms without manual intervention.

Learn more in the [setup experience guide](https://fleetdm.com/guides/macos-setup-experience).

### Android software inventory

You can now see applications installed in the work profile on personally-owned (BYOD) Android hosts. This gives you visibility into the apps users install within their managed workspace.

Learn how to turn on Android MDM features in [this guide](https://fleetdm.com/guides/android-mdm-setup).

### Lock (Lost Mode) for iOS and iPadOS

You can now remotely enable or disable [Lost Mode](https://support.apple.com/guide/security/managed-lost-mode-and-remote-wipe-secc46f3562c/web#:~:text=locked%20or%20erased.-,Managed%20Lost%20Mode,-If%20a%20supervised) on company-owned iPhones and iPads. In Fleet, head to the host's **Host details page** and select **Actions > Lock**. If a host goes missing, you can lock it down fast and protect sensitive data.

### New Fleet-maintained apps

Fleet added [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) for Cursor, 010 Editor, and Linear on macOS and Cursor on Windows. See all Fleet-maintained apps in the [software catalog](https://fleetdm.com/software-catalog).

## Changes

### Security Engineers
- Added support for software inventory on Android hosts.
- Added support for npm packages in software inventory and vulnerability matching for macOS and Linux hosts.
- Added support for JetBrains inventory on hosts.
- Added vulnerbaility detection in JetBrains plugins.
- Added support for VSCode fork (Cursor, Windsurf, VSCodium, VSCodium Insiders, and Trae) extensions in software inventory. 
- Added Santa tables to fleetd.

### IT Admins
- Added ability to install software for iOS and iPadOS hosts during the setup experience.
- Added ability to specify VPP apps for automatic installation during ADE iOS and iPadOS host enrollment.
- Added the ability to lock iOS and iPadOS devices through lost mode.
- Added support for locking and unlocking iOS and iPadOS devices from the UI.
- Added configuration option to setup experience for macOS hosts to halt if any software install fails.
- Added `gigs_all_disk_space` vital collection, storage, service, and UI rendering for Linux hosts.
- Added new server config flag for specifying the cleanup age for completed distributed targets.

### Other improvements and bug fixes
- Added link component shown in the host column to the host details page.
- Added flash warning when an unauthorized user tries to access teams settings.
- Added descriptive error in cases of manual MacOS profile download failure. 
- Updated the MacOS setup experience to use the new web UI.
- Updated the UI for adding new scripts to the scripts library.
- Changed display logic for the organization logo component on the My Device page to prevent flickering.
- Improved performance of `/api/latest/fleet/os_versions` endpoint, especially for deployments with Linux hosts.
- Optimized MySQL queries on `/api/latest/fleet/vulnerabilities` and `/api/latest/fleet/software/versions` to improve performance for Fleet UI use cases.
- Optimized `/config` API endpoint to use the primary DB node for both persisting changes and fetching modified app config.
- Improved live query response times by adding a new server config flag for specifying the cleanup age for completed distributed targets.
- Improved query performance by using a lighter-weight query for checking if a team is enabled for conditional access.
- Changed license warning to only show one time during GitOps runs.
- Updated to allow setting an org support url to use the "file" protocol in the url.
- Changed the default name of Host Identity CA to 'Fleet Host Identity CA' to avoid conflict with Fleet's Apple MDM CA.
- Updated host details run script user flows to include a confirmation step.
- Applied singular word form to GitOps log messages when a single entity is referenced in the message.
- Updated the "Setting up your device" page to show status of setup script run.
- Deprecate `browser` in favor of `extension_for` in API responses and JSON/YAML outputs.
- Added migration to clear the `platform` field on all _builtin_ labels.
- Added migration to relink missing SCIM user data to hosts.
- Updated host certificate renewal flow for NDES, Smallstep, custom scep proxy CAs to support $FLEET_VAR_SCEP_RENEWAL_ID in the OU field rather than CN.
- Updated device mapping API to allow an "idp" source to manually set IDP user mappings.
- Updated styling to be more consistent in edit policies view for FireFox.
- Replaced outdated Firefox icon with a new one that follows brand guidelines.
- Allowed testing a new or edited policy query via live query while in GitOps Mode.
- Fixed missing "failed" VPP app install activities when installation is canceled due to MDM being turned off for a host.
- Fixed bug where uploading a software installer failed because it was "not found in the datastore".
- Fixed missing aboslute timestamp tooltips on script creation date in script list, query modification date in query list.
- Fixed bug with the ChangeManagement component where the GitOps checkbox local UI state was being reset due to GET request after PATCH request.
- Fixed MySQL deadlocks when multiple hosts are updating their certificates in host vitals at the same time.
- Fixed an issue where longer variable names ($FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART) with the same base ($FLEET_VAR_HOST_END_USER_IDP_USERNAME) was not processed in the right order.
- Fixed UI bug where "Show disk encryption key" option was incorrectly displayed for hosts enrolled with a third-party MDM solution.
- Fixed WhatsApp and VS Code icons not displaying correctly
- Fixed bad software ingestion debug message and added filter for invalid software with missing names.
- Fixed a bug where a software installer could be installed in the same team and same platform (macOS) where an App Store app already existed for the same software title, and vice-versa (App Store app added when a sofware package already existed, this one was only possible just via `fleetctl gitops`).
- Fixed listing hosts with `populate_software` not returning hash_sha256 for macos apps.
- Fixed bug where batch setting MDM profiles could cause a nil pointer dereference when processing an invalid profile (e.g., cannot parse mobileconfig because it is bad xml).
- Fixed bug hiding the UI elements post install script output in Software Install Details modal.
- Fixed software title host count mismatch that was caused by including software installers in the count.
- Fixed a scenario where a wiped Windows host re-enrolled as a distinct host row in Fleet and the previous host's page could not be loaded successfully.
- Fixed an issue where a host transfer on `mdm_enrolled` activity would be reversed by orbit enroll.
- Fixed a bug in live queries that caused `livequery:{$CAMPAIGN_ID}` Redis keys to not be cleaned up or expire.
- Fixed inconsistency in GitOps for App store apps if no VPP token was found, so that both dry run and actual run fails.
- Fixed the software title counts by status to be consistent with the status reported in the host's software list and filter by status.
- Fixed outdated tooltip on dark background logo URL field in Organization info settings.
- Fixed `fleetctl generate-gitops` when MDM is not turned on.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.76.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-11-07">
<meta name="articleTitle" value="Fleet 4.76.0 | Self-service scripts, JetBrains/Cursor/Windsurf vulnerabilities, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.76.0-1600x900@2x.png">
