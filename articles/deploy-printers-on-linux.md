# Deploy printers on Linux with Fleet

You can deploy printers to Linux hosts with the same self-service scripts you'd use for any other software. Most Linux desktop distributions use CUPS, so the approach mirrors macOS.

## Prerequisites

- Fleet, with Linux hosts enrolled.
- Admin or maintainer access to the fleet you're deploying to.
- `fleetd` deployed with [scripts enabled](https://fleetdm.com/guides/scripts#enable-scripts).
- [Fleet Premium](https://fleetdm.com/pricing) to offer the printer in self-service or scope it to a subset of hosts with labels. Running a script on demand or via policy automation works on Fleet Free.
- `cups-client` (Debian, Ubuntu), `cups` (Fedora, RHEL, and Arch), or the CUPS package for your distribution installed on your hosts. It's not guaranteed to ship by default, so confirm it before deploying the script below.
- The printer's hostname or IP address, and the queue or resource path it prints to.

## Offer a printer in self-service

Add the printer as a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages):

1. Go to **Software**, choose a fleet, and select **Add software > Custom package**.
2. Choose a `.sh` file containing the script below.
3. Under **Advanced options**, add the uninstall script.
4. Check **Self-service**, then select **Add software**.

```sh
#!/bin/sh

PRINTER_NAME="Floor2-LaserJet"
PRINTER_LOCATION="Floor 2"
PRINTER_URI="ipp://192.0.2.10/ipp/print"

lpadmin -p "$PRINTER_NAME" -L "$PRINTER_LOCATION" -E -v "$PRINTER_URI" -m everywhere
```

`-m everywhere` uses IPP Everywhere's driverless printing, which covers most current network printers without a PPD. If the printer doesn't support it, stage the manufacturer's PPD on the host first: add a second software package containing the PPD as a `.tar.gz` archive, with an install script like `cp "$INSTALLER_PATH"/*.ppd /usr/share/cups/model/printer.ppd`, and deploy it as an automatic install. Then, in the printer script above, replace `-m everywhere` with `-P /usr/share/cups/model/printer.ppd`.

Uninstall script:

```sh
#!/bin/sh

lpadmin -x "Floor2-LaserJet"
```

Repeat this for each printer, using a distinct `PRINTER_NAME` and file name each time. End users see every printer you mark self-service on the **Fleet Desktop > Self-service** page and pick the one for their location.

In GitOps, add the script directly as a script-only package:

```yaml
software:
  packages:
    - path: ../lib/linux/scripts/floor2-laserjet.sh
      display_name: Floor 2 printer
      self_service: true
```

> **Note:** GitOps doesn't currently expose a field for an uninstall script on script-only packages. If you need one, set it once from the Fleet UI after the package exists; re-running GitOps won't remove it.

## Verify

1. On the host's **Host details** page in Fleet, open the **Activity** tab and confirm the script install shows a success status.
2. On the host, check the printer queue with `lpstat -p`, or open your distribution's printer settings panel.
3. Print a test page from the host to confirm the connection works, not just that the queue was created.

## Troubleshoot

**The script succeeds but the printer doesn't appear**

`lpadmin` returns success even when the connection URI is wrong, since it only creates the queue. Confirm the IP address, hostname, and resource path against the printer's own network settings page.

**The script fails with "lpadmin: command not found"**

`cups-client` or `cups` isn't installed on the host. Install it first, either as a prerequisite software package or as the first step of the install script, before running `lpadmin`.

See [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet) for the same task on other platforms.

<meta name="articleTitle" value="Deploy printers on Linux with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Deploy a printer to Linux hosts with a self-service script using CUPS and lpadmin.">
