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

With Fleet, you can now use a new "include any" label option to target OS settings (configuration profiles) to specific hosts within a team. This added flexibility allows for finer control over which OS settings apply to which hosts, making it easier to tweak configurations without disrupting broader baselines (aka Fleet [teams](https://fleetdm.com/guides/teams#basic-article)).

### Preview scripts before run

Fleet now provides the ability to preview scripts directly on the **Host details** or **Scripts** page. This quick-view feature reduces the risk of errors by letting you verify the script is correct before running it, saving time and ensuring smoother operations.

## Changes

### Device management (MDM)
- Fixed a bug where users could attempt to install an App Store app on a host that was not MDM enrolled.
- Fixed MDM configuration profiles deployment when based on excluded labels.
- Dismissed error flash on the my device page when navigating to another URL.
- Fixed an issue where the create and update label endpoints could return outdated information in a deployment using a MySQL replica.
- Added indicator of how fresh a software title's host and version counts are on the title's details page.
- Reboot linux machine on unlock to work around GDM bug on Ubuntu.
- Cancelled pending script executions when a script is edited or deleted.
- Fix some cases where Fleet Maintained Apps generated incorrect uninstall scripts.

### Observability
- Users are now prompted to reenter the password in the Fleet UI if SCEP/NDES URL or username has changed.
- Users can now view scripts in the UI without downloading them.
- Creating a query now allows users to turn on/off automations transparently regarding the current log destination.
- Fixed path resolution for installer queries and scripts to always be relative to where the query file or script is referenced. This change may break existing YAML files that had to account for previous inconsistent behavior.
- Added support for deb packages compressed with zstd.
- Updated GitOps to return an error if the deprecated `apple_bm_default_team` key is used and there are more than 1 ABM tokens in Fleet.
- Add UI for allowing users to install custom profiles on hosts that include any of the defined labels.
- Improved memory usage of the Fleet server when uploading a large software installer file.

### Software management
- Fixed issue with uploading macOS software packages without a top level Distribution.xml but with a top level PackageInfo.xml.
- Added better handling of timeout and insufficient permissions errors in NDES SCEP proxy.
- Allowed skipping computationally heavy population of vulnerability details when populating host software on hosts list endpoint when using Fleet Premium.

### Bug fixes and improvements
- Set a more elegant minimum height for the Add hosts > ChromeOS > Policy for extension field, avoiding a scrollbar.
- Fixed a bug where the software batch endpoint status code was updated from 200 (OK) to 202 (Accepted).
- Added capability for Fleet to serve yara rules to agents over HTTPS authenticated via node key.
- Fixed a bug in the software batch endpoint status code.
- Generate an activity when activity automations are enabled, edited, or disabled.
- Major improvements to keyboard accessibility throughout the Fleet UI.
- Updated a package used for testing (msw) to improve security.
- Added support for "include any" label/profile relationships to the profile reconciliation machinery.
- Added DB support for "include any" label profile deployment.
- Added support for labels_include_any to GitOps.
- Added info banner for cloud customers to help with their windows autoenrollment setup.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.60.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2024-11-25">
<meta name="articleTitle" value="Fleet 4.60.0 | Escrow Linux disk encryption keys, custom targets for OS settings, scripts preview">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.60.0-1600x900@2x.png">