# Fleet 4.90.0 | Windows account controls, Android vulnerability visibility, and full DDM support

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.90.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.90.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Windows: force a standard end user account on automatic enrollment](#windows-force-a-standard-end-user-account-on-automatic-enrollment)
- [Windows: custom BitLocker configuration profiles](#windows-custom-bitlocker-configuration-profiles)
- [Android: OS versions and vulnerabilities on Software > OS](#android-os-versions-and-vulnerabilities-on-software-os)
- [Custom host vitals for every platform, including mobile](#custom-host-vitals-for-every-platform-including-mobile)
- [Rename macOS, iOS, and iPadOS hosts](#rename-macos-ios-and-ipados-hosts)
- [Support for all Apple DDM profiles and assets](#support-for-all-apple-ddm-profiles-and-assets)
- [Edit a configuration profile's labels or contents without deleting it](#edit-a-configuration-profiles-labels-or-contents-without-deleting-it)
- [macOS local account creation and password sync with any IdP](#macos-local-account-creation-and-password-sync-with-any-idp)
- [Upload multiple custom packages for the same software title](#upload-multiple-custom-packages-for-the-same-software-title)
- [New log destination: Splunk](#new-log-destination-splunk)

### Windows: force a standard end user account on automatic enrollment

IT admins enrolling Windows hosts automatically without Autopilot device registration can now force the end user's account to be created as a standard (non-admin) account. Fleet creates the local admin account first, so the end user's account comes up standard by default, the same outcome organizations already get today when enrolling through Autopilot. This closes the gap for IT admins who haven't set up Autopilot device registration but still want end users on standard accounts from the first login.

GitHub issue: [#43490](https://github.com/fleetdm/fleet/issues/43490)

### Windows: custom BitLocker configuration profiles

IT admins can now upload a custom BitLocker configuration profile for Windows hosts, using the same `FLEET_MDM_ENABLE_CUSTOM_FILEVAULT` environment variable that already unlocks custom FileVault profiles for macOS (also accepted as `FLEET_MDM_ENABLE_CUSTOM_DISK_ENCRYPTION`). This means Windows disk encryption can be customized beyond Fleet's built-in BitLocker controls, matching the flexibility already available for macOS.

GitHub issue: [#43518](https://github.com/fleetdm/fleet/issues/43518)

### Android: OS versions and vulnerabilities on Software > OS

The **Software > OS** page now shows Android OS versions and their known vulnerabilities alongside every other platform Fleet tracks. Security engineers get one place to see OS-level exposure across a fleet that includes personally-owned (BYOD) Android hosts, with the Android security update version formatted as a date (for example, `2026-07-01`) for easier tracking against Google's monthly security bulletins.

GitHub issue: [#35075](https://github.com/fleetdm/fleet/issues/35075)

### Custom host vitals for every platform, including mobile

IT admins can now define custom host vitals, like an asset tag or a Jamf device ID, for any host platform, including iOS, iPadOS, and Android, not just macOS, Windows, and Linux. Custom vitals show up on the host details page and can be used as variables in scripts and configuration profiles, so values that Fleet can't discover on its own (because they live in another system) can still drive automation everywhere in your fleet.

GitHub issue: [#44954](https://github.com/fleetdm/fleet/issues/44954)

### Rename macOS, iOS, and iPadOS hosts

IT admins can now set a host name template on the **Controls** page for a fleet (or "No team") that applies to macOS, iOS, and iPadOS hosts. This gives every Apple host a standard naming convention, often including its serial number, without having to build a custom automation that sends an MDM rename command to each device.

GitHub issue: [#38806](https://github.com/fleetdm/fleet/issues/38806)

### Support for all Apple DDM profiles and assets

Fleet now supports uploading any Apple declarative device management (DDM) configuration or asset type, for both the device and user channel, instead of a limited set. IT admins can deploy DDM profiles like the Safari extensions settings declaration and reference DDM assets from those profiles, so Fleet keeps pace with new declaration types as Apple ships them instead of admins waiting on Fleet to add each one individually.

GitHub issue: [#38986](https://github.com/fleetdm/fleet/issues/38986)

### Edit a configuration profile's labels or contents without deleting it

IT admins can now edit a configuration profile's labels (switching between include any, include all, and exclude any) or its contents directly from **Controls > OS settings > Configuration profiles**, without deleting and re-uploading it. This works across macOS, Windows, Android, and Apple DDM (declaration) profiles, and it makes staged rollouts easier since editing a profile no longer removes it from the hosts that already have it.

GitHub issue: [#38869](https://github.com/fleetdm/fleet/issues/38869)

### macOS local account creation and password sync with any IdP

_Experimental, available in Fleet Premium_

During automated (ADE) enrollment, Fleet can now create the end user's macOS local account and keep its password in sync with any identity provider that supports OAuth Resource Owner Password Grant (ROPG), not just Okta or Entra's native Platform SSO integrations. A new Fleet-built Platform SSO extension proxies authentication through the Fleet server, so end users get one password, meeting your organization's requirements, to unlock their Mac, their keychain, and third-party tools.

GitHub issue: [#45524](https://github.com/fleetdm/fleet/issues/45524)

### Upload multiple custom packages for the same software title

IT admins can now upload up to 10 custom packages for the same software title in the same fleet. This makes it possible to deploy different versions or architectures, like Arm versus Intel builds, or run staged rollouts, using labels to target the right package instead of maintaining separate fleets for each variant.

GitHub issue: [#28108](https://github.com/fleetdm/fleet/issues/28108)

### New log destination: Splunk

Fleet can now send reports and other osquery logs directly to Splunk over Splunk's HTTP Event Collector (HEC), without setting up Firehose as a middleman first.

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
