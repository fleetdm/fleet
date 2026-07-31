# Fleet 4.90.0 | Windows BitLocker controls, Android vulnerabilities, full DDM profile support, and more...

Fleet 4.90.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.90.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Windows: custom BitLocker configuration profiles](#windows-custom-bitlocker-configuration-profiles)
- [Android: OS versions and vulnerabilities on Software > OS](#android-os-versions-and-vulnerabilities-on-software-os)
- [Support for all DDM profiles and assets](#support-for-all-ddm-profiles-and-assets)
- [Custom host vitals for every platform](#custom-host-vitals-for-every-platform)
- [Rename macOS, iOS, and iPadOS hosts](#rename-macos-ios-and-ipados-hosts)
- [macOS local account creation and password sync with any IdP](#macos-local-account-creation-and-password-sync-with-any-idp)
- [Edit a configuration profile's labels or contents without deleting it](#edit-a-configuration-profiles-labels-or-contents-without-deleting-it)
- [Upload multiple custom packages for the same software title](#upload-multiple-custom-packages-for-the-same-software-title)
- [New log destination: Splunk](#new-log-destination-splunk)

### Windows: custom BitLocker configuration profiles

IT Admins can now upload a custom BitLocker configuration profile for Windows hosts, using  [`FLEET_MDM_ENABLE_CUSTOM_DISK_ENCRYPTION` server configuration option](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-enable-custom-disk-encryption). This means Windows disk encryption can be customized beyond Fleet's built-in BitLocker controls, matching the flexibility already available for macOS.

Only self-managed users and customers can modify Fleet server configuration. If you're a managed-cloud customer, please reach out to Fleet about modifying the configuration.

For users that already have `FLEET_MDM_ENABLE_CUSTOM_FILEVAULT` enabled, no changes are necessary. Fleet just added a second, cross-platform name for this key.

GitHub issue: [#43518](https://github.com/fleetdm/fleet/issues/43518)

### Android: OS versions and vulnerabilities

The **Software > OS** page now shows Android OS versions and their known vulnerabilities alongside every other platform Fleet tracks. Security Engineers get one place to see OS-level exposure across a fleet that includes personally-owned (BYOD) Android hosts, with the Android security update version formatted as a date (for example, `2026-07-01`) for easier tracking against Google's monthly security bulletins.

GitHub issue: [#35075](https://github.com/fleetdm/fleet/issues/35075)

### Support for all DDM profiles and assets

Fleet now supports uploading any Apple declarative device management (DDM) configuration or asset, for both the device and user channel. IT Admins can deploy DDM profiles like the Safari extensions settings declaration and reference DDM assets from those profiles. This way, anytime Apple ships new DDM features, IT Admins can use them on day one.

GitHub issue: [#38986](https://github.com/fleetdm/fleet/issues/38986)

### Custom host vitals for every platform

IT Admins can now define custom host vitals, like an asset tag or warranty expiration, for all platforms (macOS, Windows, Linux, iOS/iPadOS, and Android). Custom vitals appear on the **Host details** page and can be used to create labels and as variables in scripts and configuration profiles, so values from another system can drive automation everywhere in your fleet. [Learn more](https://fleetdm.com/guides/articles/custom-host-vitals).

GitHub issue: [#44954](https://github.com/fleetdm/fleet/issues/44954)

### Rename macOS, iOS, and iPadOS hosts

IT Admins can now set a host name template on the **Controls** page for a fleet that applies to macOS, iOS, and iPadOS hosts. This gives every Apple host a standard naming convention, often including its serial number, without having to build a custom automation that sends an MDM rename command to each host. [Learn how](https://fleetdm.com/guides/articles/rename-hosts-with-a-naming-template).

GitHub issue: [#38806](https://github.com/fleetdm/fleet/issues/38806)

### macOS local account creation and password sync with any IdP

During automated (ADE) enrollment, Fleet can now create the end user's macOS local account and keep its password in sync with any identity provider that supports OAuth Resource Owner Password Grant (ROPG) (e.g. Okta). End users get one password, meeting your organization's requirements, to unlock their Mac, their keychain, and third-party tools. [Learn more](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet).

The [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin) is required for local account creation and password sync. Add the app from the Fleet-maintained catalog and configure it to install during new Mac setup. [Learn how](https://fleetdm.com/guides/setup-experience#install-software).

GitHub issue: [#45524](https://github.com/fleetdm/fleet/issues/45524)

### Edit a configuration profile's labels or contents without deleting it

IT Admins can now edit a configuration profile. This includes the profile's labels (switching between include any, include all, and exclude any) or its contents directly from **Controls > OS settings > Configuration profiles**, without deleting and re-uploading it. This works across macOS, Windows, and Android profiles, and it makes staged rollouts easier since editing a profile no longer requires removing the older version from the hosts that already have it.

GitHub issue: [#38869](https://github.com/fleetdm/fleet/issues/38869)

### Upload multiple custom packages for the same software title

IT Admins can now upload up to 10 custom packages for the same software title in the same fleet. This makes it possible to deploy different versions or architectures, like Arm versus Intel builds, or run staged rollouts, using labels to target the right package instead of maintaining separate fleets for each variant.

GitHub issue: [#28108](https://github.com/fleetdm/fleet/issues/28108)

### New log destination: Splunk

Fleet can now send reports and other osquery logs directly to Splunk, without setting up Firehose as a middleman first. Learn how to [send reports directly to Splunk](https://fleetdm.com/guides/log-destinations#splunk).

GitHub issue: [#26333](https://github.com/fleetdm/fleet/issues/26333)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.90.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-07-30">
<meta name="articleTitle" value="Fleet 4.90.0 | Windows account controls, Android vulnerability visibility, and full DDM support">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.90.0-1600x900@2x.png">
