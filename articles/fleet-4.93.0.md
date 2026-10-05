# Fleet 4.93.0 | macOS app patching, Android zero-touch enrollment, Windows admin password rotation, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/0qYhQAycHu0?si=DoIXdNTs-R-1M7p4" allowfullscreen></iframe>
</div>
-->

Fleet 4.93.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.93.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [macOS app patching: prompt end users before deadline](#macos-app-patching-prompt-end-users-before-deadline)
- [Android zero-touch enrollment](#android-zero-touch-enrollment)
- [More Android host vitals](#more-android-host-vitals)
- [Rotate Windows local admin password](#rotate-windows-local-admin-password)
- [Restrict Managed Apple Account sign-in to managed devices](#restrict-managed-apple-account-sign-in-to-managed-devices)
- [Hardware attestation for iOS and iPadOS](#hardware-attestation-for-ios-and-ipados)
- [Reports that cover every host](#reports-that-cover-every-host)

### macOS app patching: prompt end users before deadline

_Available in Fleet Premium_

IT admins can now warn end users before Fleet patches an open macOS app. When a patch policy with `notify_before_patching` fails and the app is open, Fleet Desktop shows a notification listing the apps that will update, waits one hour, shows a reminder five minutes before, and then installs the update. This gives end users time to save their work, instead of waiting until they close the app on their own (`patch_when_closed`).

The deadline is one hour. Setting a custom deadline is [coming soon](https://github.com/fleetdm/fleet/issues/39176).

See what end users experience, including a video walkthrough, in the [patching end user experience guide](https://fleetdm.com/guides/patching-end-user-experience).

GitHub issue: [#39178](https://github.com/fleetdm/fleet/issues/39178)

### Android zero-touch enrollment

_Available in Fleet Premium_

IT admins can now ship company-owned Android devices straight to end users and have them enroll in Fleet the first time they're turned on. In **Settings > Integrations > MDM > Android zero-touch**, copy the DPC extras JSON into Google's zero-touch portal. Zero-touch-enrolled hosts land in "Unassigned." Assigning them to a different fleet automatically is [coming soon](https://github.com/fleetdm/fleet/issues/51479).

Learn how to set it up in the [Android zero-touch enrollment guide](https://fleetdm.com/guides/android-zero-touch-enrollment).

GitHub issue: [#49165](https://github.com/fleetdm/fleet/issues/49165)

### More Android host vitals

IT admins can now see more Android host vitals on the **Host details** page, including whether USB debugging is on, whether a passcode is set, whether Google Play Protect is on, encryption status, security patch level, manufacturer, security posture (from Google's Play Integrity checks), and phone numbers. These vitals are also returned by the [get host API](https://fleetdm.com/docs/rest-api/rest-api#get-host).

GitHub issue: [#49791](https://github.com/fleetdm/fleet/issues/49791)

### Rotate Windows local admin password

_Available in Fleet Premium_

IT admins can now rotate the password for Fleet's managed local admin account on Windows hosts, the same way they already could on macOS. Go to **Host details > Actions > Show managed account** and select **Rotate password**. Fleet also rotates the password automatically an hour after someone views it, so a password shared for troubleshooting doesn't stay valid.

Learn more about the [managed local account on Windows](https://fleetdm.com/guides/windows-linux-setup-experience#managed-local-account-windows).

GitHub issue: [#43489](https://github.com/fleetdm/fleet/issues/43489)

### Restrict Managed Apple Account sign-in to managed devices

_Available in Fleet Premium_

IT admins can now make sure end users sign in to their Managed Apple Account only on devices enrolled in Fleet. This stops company data from syncing to personal, unmanaged Apple devices. In Apple Business, set **Allow Managed Apple Account on** to **Managed devices only** or **Supervised devices only**, and Fleet confirms to Apple that the host is managed during sign-in. 

Hosts that automatically enroll (ADE) are already tied to an Apple Business (AB). Manually enrolled hosts aren't, so Fleet uses your default AB for sign-in. If you've added more than one AB, choose a default for sign-in in **Settings > Integrations > MDM > Apple Business (AB)**. Otherwise, Managed Apple Account sign-in fails on manually enrolled hosts.

Learn more in the [Apple MDM setup guide](https://fleetdm.com/guides/apple-mdm-setup#restrict-apple-account-sign-in-managed-apple-accounts).

> Only turn on this setting if you're just starting to roll out Managed Apple Account sign-in. If your end users already sign in with Managed Apple Accounts, leave it off for now. Hosts enrolled before Fleet 4.93.0 can't sign in to Managed Apple Accounts with this setting on until their next enrollment profile renewal, about every six months.
>
> If you want to turn it on sooner, follow [these instructions](https://docs.google.com/document/d/1f3OZaC9lhN58esD3cXzqEePx2kq_tHX7b1KgwSEOQ3c/edit?tab=t.0).

GitHub issue: [#45829](https://github.com/fleetdm/fleet/issues/45829)

### Hardware attestation for iOS and iPadOS

_Available in Fleet Premium_

Hardware attestation (ACME), which Fleet already supports for Apple silicon Macs, now works for iPhones and iPads assigned to Fleet in Apple Business. With `apple_require_hardware_attestation` on, iPhones and iPads from 2017 or later (with an A11 Bionic chip or later), running iOS or iPadOS 16 or later, prove their hardware matches a known Apple Business record when they enroll. Hosts already enrolled with SCEP move to ACME on their next certificate renewal. Older devices keep enrolling with SCEP.

Learn more in the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files#controls).

GitHub issue: [#51528](https://github.com/fleetdm/fleet/issues/51528)

### Reports that cover every host

Reports now store as many results as you have hosts, instead of stopping at 1,000. A report that returns one result per host covers your whole fleet, so you can bring Jamf extension attributes over to Fleet. Sorting, search, and pagination now run on the server, so large reports stay usable.

Fleet doesn't store a host's result over 512 KB. Learn more in the [reports guide](https://fleetdm.com/guides/reports).

GitHub issue: [#43723](https://github.com/fleetdm/fleet/issues/43723)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.93.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-10-02">
<meta name="articleTitle" value="Fleet 4.93.0 | macOS app patching, Android zero-touch enrollment, Windows admin password rotation, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.93.0-1600x900@2x.png">
