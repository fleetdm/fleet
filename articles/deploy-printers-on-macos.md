# Deploy printers on macOS with Fleet

You can deploy printers to macOS hosts with the same scripts and configuration profiles you'd use for any other software or setting. This guide covers two approaches: a self-service script that lets each end user install the printer for their location, and a configuration profile that installs a fixed printer list on every host.

## Prerequisites

- Fleet, with macOS hosts enrolled.
- Admin or maintainer access to the fleet you're deploying to.
- `fleetd` deployed with [scripts enabled](https://fleetdm.com/guides/scripts#enable-scripts). This happens automatically for hosts with MDM turned on.
- [Fleet Premium](https://fleetdm.com/pricing) to offer the printer in self-service or scope it to a subset of hosts with labels. Running a script on demand or via policy automation works on Fleet Free.
- The printer's hostname or IP address, and the queue or resource path it prints to.

## Offer a printer in self-service

Add the printer as a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages):

1. Go to **Software**, choose a fleet, and select **Add software > Custom package**.
2. Choose a `.sh` file containing the script below.
3. Under **Advanced options**, add the uninstall script.
4. Check **Self-service**, then select **Add software**.

The script adds the printer with [`lpadmin`](https://www.cups.org/doc/man-lpadmin.html), using IPP Everywhere's driverless mode, which covers most current network printers without a manufacturer PPD.

```sh
#!/bin/sh

PRINTER_NAME="Floor2-LaserJet"
PRINTER_LOCATION="Floor 2"
PRINTER_URI="ipp://192.0.2.10/ipp/print"

/usr/sbin/lpadmin -p "$PRINTER_NAME" -L "$PRINTER_LOCATION" -E -v "$PRINTER_URI" -m everywhere -o printer-is-shared=false
```

> **Note:** If the printer doesn't support IPP Everywhere, stage the manufacturer's PPD on the host first. Add a second software package containing the PPD as a `.tar.gz` archive, with an install script like `cp "$INSTALLER_PATH"/*.ppd /Library/Printers/PPDs/Contents/Resources/printer.ppd`, and deploy it as an automatic install. Then, in the printer script above, replace `-m everywhere` with `-P /Library/Printers/PPDs/Contents/Resources/printer.ppd`. Apple's own CUPS project has been [deprecating PPD-based printer drivers](https://www.cups.org/blog/2018-06-06-demystifying-cups-development.html) since 2018 in favor of driverless printing, so treat this as a fallback for older printers rather than the default.

Uninstall script:

```sh
#!/bin/sh

/usr/sbin/lpadmin -x "Floor2-LaserJet"
```

Repeat this for each printer, using a distinct `PRINTER_NAME` and file name each time. End users see every printer you mark self-service on the **Fleet Desktop > Self-service** page and pick the one for their location.

In GitOps, add the script directly as a script-only package:

```yaml
software:
  packages:
    - path: ../lib/macos/scripts/floor2-laserjet.sh
      display_name: Floor 2 printer
      self_service: true
```

> **Note:** GitOps doesn't currently expose a field for an uninstall script on script-only packages. If you need one, set it once from the Fleet UI after the package exists; re-running GitOps won't remove it.

## Add the printer

Use this when every host on a fleet needs the same printer, with no end user choice involved.

1. Build a Printing (`com.apple.mcxprinting`) payload in [iMazing Profile Creator](https://imazing.com/profile-editor), starting from this [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/printer.mobileconfig). For each printer, set the display name, connection URI, and PPD. Or skip the GUI and have an AI coding agent generate and validate the payload for you. See [Build and validate configuration profiles with AI instead of a GUI](https://fleetdm.com/guides/build-configuration-profiles-with-ai).
2. In Fleet, go to **Controls > OS settings > Configuration profiles** and choose a fleet.
3. Select **Add profile** and upload the `.mobileconfig` file.

> **Warning:** [Apple allows only one Printing payload](https://support.apple.com/guide/deployment/printing-payload-settings-dep9514788c/web) per host. If you need more than one printer, add every printer to the same profile rather than uploading separate profiles, or the second upload replaces the first.

> **Note:** Configuration profiles push to every targeted host today, so this option can't offer a choice of printer the way the self-service script above does. [Self-service configuration profiles](https://github.com/fleetdm/fleet/issues/46834) are planned for a future release of Fleet, after which end users will be able to opt in to a printer profile instead of having it pushed to them.

In GitOps:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/profiles/floor2-printer.mobileconfig
```

## Verify

1. On the host's **Host details** page in Fleet, open the **Activity** tab and confirm the script or profile install shows a success status.
2. On the host, open **System Settings > Printers & Scanners** and confirm the printer appears.
3. Print a test page from the host to confirm the driver and connection both work, not just that the queue was created.

## Troubleshoot

**The script succeeds but the printer doesn't appear**

`lpadmin` returns success even when the connection URI is wrong, since it only creates the queue. Confirm the IP address, hostname, and resource path against the printer's own network settings page.

**The printer doesn't support IPP Everywhere**

`-m everywhere` fails, or the queue installs but prints garbled output, on printers old enough to lack IPP Everywhere support. Stage the manufacturer's PPD on the host and use `-P` instead, following the note under the script above.

**The configuration profile doesn't add a second printer**

Only one Printing payload can exist per host. If you added a second profile instead of adding the printer to the existing one, delete the second profile and add all printers to a single payload.

See [Configuration profiles](https://fleetdm.com/guides/custom-os-settings) for how Fleet delivers and verifies profiles, and [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet) for the same task on other platforms.

<meta name="articleTitle" value="Deploy printers on macOS with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Deploy printers to macOS hosts with a self-service script, or install one with a configuration profile.">
