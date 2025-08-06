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

Fleet Desktop now shows only the relevant software actions (install, update, uninstall) based on the actual state of the app on each machine. End users see exactly what they can do—nothing more, nothing less. Learn more in [the guide](https://fleetdm.com/guides/updating-software-in-fleet-admin-and-fleet-desktop-workflows).

### More Fleet-maintained apps

You can now manage these popular apps as Fleet-maintained software—no need to hunt down vendor installers or build packages yourself. Just select and deploy. Learn more about [Fleet-mainted apps](https://fleetdm.com/guides/fleet-maintained-apps).

### Linux host identity certificates

Fleet now supports TPM-based identity for Linux hosts. When you deploy [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd), it can automatically obtain a hardware-backed identity certificate (similar to macOS MDM). This certificate is required to communicate with the Fleet server—enhancing trust and tamper resistance for your Linux fleet.

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.72.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-08-08">
<meta name="articleTitle" value="Fleet 4.72.0 | Account-based user enrollment, smarter self-service, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.72.0-1600x900@2x.png">
