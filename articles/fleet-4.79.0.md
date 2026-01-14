# Fleet 4.79.0 | Latest macOS only at enrollment, MDM command history, and more...

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.79.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.79.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Latest macOS only at enrollment
- MDM command history
- Android software configuration
- Cross-platform certificate deployment

### Latest macOS only at enrollment

You can now choose to enforce the latest macOS version only for newly enrolling hosts. Already enrolled hosts are unaffected. This gives you room to manage updates with tools like Nudge instead.

### MDM command history

On the **Host details** page for macOS, iOS, and iPadOS hosts, you’ll now see a list of all MDM commands: past and upcoming. This includes both Fleet-initiated commands and those triggered by an IT admins. This can help you and your support team troubleshoot why hosts are missing configuration profiles or certificates.

### Android software configuration

IT Admins can now configure Android software (`managedConfiguration`) directly in the Fleet UI, via GitOps (YAML), or via Fleet's API. This makes it easier to customize behavior across your Android hosts. For example, you can configure the default `portal` for Palo Alto's [GlobalProtect app](https://docs.paloaltonetworks.com/globalprotect/administration/globalprotect-apps/deploy-the-globalprotect-app-on-mobiles/manage-the-globalprotect-app-using-other-third-party-mdms/configure-the-globalprotect-app-for-android) to automatically navigate end users to your VPN. Learn more about configuring Android software in [the guide](TODO).

### Cross-platform certificate deployment

You can now install certificates from any [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificate authority on corporate Android devices. This helps you connect your end users to Wi-Fi, VPN, and other tools.

This means you can now install certificates on Android, macOS, Windows, Linux, and iOS/iPadOS hosts. See all certificate authorities supported by Fleet in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.79.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-01-14">
<meta name="articleTitle" value="Fleet 4.79.0 | TODO">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.79.0-1600x900@2x.png">
