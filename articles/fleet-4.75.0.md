# Fleet 4.75.0 | Omarchy Linux, Android configuration profiles, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/a6qx4th3dKs?si=KYVIvqZTb9AZM27Y" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.75.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.75.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Omarchy Linux
- Android configuration profiles
- Smallstep certificates
- Labels page
- Easy-to-read MDM commands

### Omarchy Linux

Fleet now supports [Omarchy](https://omarchy.org/) Linux. You can view host vitals like software inventory, run scripts, and install software.

### Android configuration profiles

You can now apply custom settings to work profiles on employee-owned (BYOD) Android hosts using configuration profiles. This lets you keep Android hosts compliant and secure. Learn how to create in [this video](https://www.youtube.com/watch?v=Jk4Zcb2sR1w).

### Smallstep certificates

Fleet now integrates with [Smallstep](https://smallstep.com/) as a certificate authority. You can deliver Wi-Fi/VPN [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificates to macOS, iOS, and iPadOS hosts to automate secure network access for your end users. Learn more in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#smallstep).

### Labels page

A new **Labels** page makes it easier to view and edit labels. You can find the new **Labels** page in Fleet by selecting your avatar in the top-right corner and selecting **Labels**.

### Easy-to-read MDM commands

Long MDM payloads and outputs are now easier to read and copy, thanks to a new vertical layout in the `fleetctl get mdm command` results. Learn more about fleetctl in [the guide](https://fleetdm.com/guides/fleetctl).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.75.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-10-17">
<meta name="articleTitle" value="Fleet 4.75.0 | Omarchy Linux, Android configuration profiles, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.75.0-1600x900@2x.png">
