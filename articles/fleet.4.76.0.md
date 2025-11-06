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

### Self-service scripts

You can now create custom Linux and Windows packages that include just a script (aka payload-free packages). In Fleet, head to **Software** page and select **Add software > Custom package**. This is perfect for self-service utilities or bundling multiple scripts as part of your out-of-the-box setup experience.

### Vulnerabilities for JetBrains, Cursor, and Windsurf extensions

Vulnerabilities (CVEs) in all Cursor, Windsurf, other VSCode forks, and JetBrains IDE extensions now show up in the **Software**, **Host details**, and **My device** pages. Gain better coverage of high-risk developer tools. Learn more about CVEs in the [vulnerabilities guide](https://fleetdm.com/guides/vulnerability-processing#basic-article).

### Improved macOS, iOS, and iPadOS setup experience

During out-of-the-box macOS setup, if critical software fails to install during setup, Fleet now cancels the process and shows an error. This ensures end users run through setup again and, if they're still running into issues, contact IT before moving forward. This helps avoid misconfigured hosts in production.

For iOS and iPadoS, installing apps on company-owned iPhones and iPads during enrollment is now supported. Perfect for instantly setting up kiosk devices, shared iPads, or Zoom rooms without manual intervention.

Learn more in the [setup experience guide](https://fleetdm.com/guides/macos-setup-experience).

### Android software inventory

You can now see applications installed in the work profile on personally-owned (BYOD) Android hosts. This gives you visibility into the apps users install within their managed workspace.

Learn how to turn on Android MDM features in [this guide](https://fleetdm.com/guides/android-mdm-setup).

### Lock (Lost Mode) for iOS and iPadOS

You can now remotely enable or disable [Lost Mode](https://support.apple.com/guide/security/managed-lost-mode-and-remote-wipe-secc46f3562c/web#:~:text=locked%20or%20erased.-,Managed%20Lost%20Mode,-If%20a%20supervised) on company-owned iPhones and iPads. In Fleet, head to the host's **Host details page** and select **Actions > Lock**. If a host goes missing, you can lock it down fast and protect sensitive data.

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.76.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-10-31">
<meta name="articleTitle" value="Fleet 4.76.0 | Self-service scripts, JetBrains/Cursor/Windsurf vulnerabilities, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.76.0-1600x900@2x.png">
