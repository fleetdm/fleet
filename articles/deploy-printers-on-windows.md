# Deploy printers on Windows with Fleet

You can deploy printers to Windows hosts with the same self-service scripts you'd use for any other software. This guide covers a driverless IPP printer, which works for most current network printers with no separate driver install, and a vendor driver setup for printers that need one.

Windows has no CSP for installing a network printer with a vendor driver, aside from [Universal Print](https://learn.microsoft.com/en-us/universal-print/fundamentals/universal-print-whatis), which requires the printer to be registered with that Microsoft 365 service. Both approaches below use a script instead.

## Prerequisites

- Fleet, with Windows hosts enrolled.
- Admin or maintainer access to the fleet you're deploying to.
- `fleetd` deployed with [scripts enabled](https://fleetdm.com/guides/scripts#enable-scripts). This happens automatically for hosts with MDM turned on.
- [Fleet Premium](https://fleetdm.com/pricing) to offer the printer in self-service or scope it to a subset of hosts with labels. Running a script on demand or via policy automation works on Fleet Free.
- The printer's hostname or IP address, and the queue or resource path it prints to.

## Deploy a printer with a script

Most current printers support IPP. [`Add-Printer`](https://learn.microsoft.com/en-us/powershell/module/printmanagement/add-printer)'s `-IppURL` parameter discovers the printer directly at that URL and has Windows pick a driver for it, typically its built-in Microsoft IPP Class Driver, without installing a vendor driver yourself.

Add the printer as a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages):

1. Go to **Software**, choose a fleet, and select **Add software > Custom package**.
2. Choose a `.ps1` file containing the script below.
3. Under **Advanced options**, add the uninstall script.
4. Check **Self-service**, then select **Add software**.

```powershell
$printerName = "Floor2-LaserJet"
$printerUri = "http://192.0.2.10:631/ipp/print"

Add-Printer -Name $printerName -IppURL $printerUri
```

> **Note:** `-IppURL` and `-DriverName`/`-PortName` belong to different, mutually exclusive parameter sets on `Add-Printer`. Don't combine them, Windows selects the driver on its own once it discovers the printer at the URL.

Uninstall script:

```powershell
$printerName = "Floor2-LaserJet"

Remove-Printer -Name $printerName
```

In GitOps, add the script directly as a script-only package:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/floor2-laserjet.ps1
      display_name: Floor 2 printer
      self_service: true
```

> **Note:** GitOps doesn't currently expose a field for an uninstall script on script-only packages. If you need one, set it once from the Fleet UI after the package exists; re-running GitOps won't remove it.

## Use a vendor driver instead

If the printer doesn't support IPP, or you need the features of the manufacturer's own driver, install that driver first, then reference it by name instead of the Microsoft IPP Class Driver. This driver package is a regular [software package](https://fleetdm.com/guides/deploy-software-packages), not a script-only one, since it needs to carry the actual driver files.

1. Get the driver package from the manufacturer. You need the raw driver files (a folder containing an `.inf` file), not a self-extracting setup executable. Compress the folder into a `.tar.gz` archive.
2. Go to **Software**, choose a fleet, and select **Add software > Custom package**.
3. Choose the `.tar.gz` archive.
4. Under **Advanced options**, set the install script to the script below, then select **Add software**.

```powershell
$infPath = Get-ChildItem -Path "$env:INSTALLER_PATH" -Filter "*.inf" -Recurse | Select-Object -First 1 -ExpandProperty FullName

if (-not $infPath) {
  Write-Host "No .inf file found in driver package"
  Exit 1
}

pnputil /add-driver "$infPath" /install
Exit $LASTEXITCODE
```

This uses [`pnputil /add-driver`](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/pnputil-examples) to stage the driver into the Windows driver store.

5. After the driver installs on a test host, run [`Get-PrinterDriver`](https://learn.microsoft.com/en-us/powershell/module/printmanagement/get-printerdriver) on that host to get the exact driver name Windows registered. Vendor documentation often doesn't match this string exactly.
6. Add a second software package for the printer itself, as a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages) following the same steps as the IPP printer above, but with this script instead:

```powershell
$printerName = "Floor2-LaserJet"
$printerIp = "192.0.2.10"
$driverName = "<driver name from Get-PrinterDriver>"

Add-PrinterPort -Name $printerName -PrinterHostAddress $printerIp
Add-Printer -Name $printerName -PortName $printerName -DriverName $driverName
```

Uninstall script:

```powershell
$printerName = "Floor2-LaserJet"

Remove-Printer -Name $printerName
Remove-PrinterPort -Name $printerName
```

Deploy the driver package as an automatic install so it's on every targeted host before end users see the printer in self-service. Mark only the printer package self-service, not the driver package.

In GitOps, the driver is a regular package (hosted at a URL Fleet can download from), and the printer script is a script-only package alongside it:

```yaml
software:
  packages:
    - path: ../lib/software/floor2-printer-driver.package.yml
    - path: ../lib/windows/scripts/floor2-laserjet-vendor.ps1
      display_name: Floor 2 printer
      self_service: true
```

`lib/software/floor2-printer-driver.package.yml`:

```yaml
- url: https://example.com/drivers/floor2-printer-driver.tar.gz
  install_script:
    path: ../scripts/install-printer-driver.ps1
```

> **Note:** Fleet downloads packages defined with `url:` rather than reading them from the repo. Host the driver archive somewhere Fleet can reach it, or upload it once via the UI and reference it by `hash_sha256` instead.

## Verify

1. On the host's **Host details** page in Fleet, open the **Activity** tab and confirm the script or package install shows a success status.
2. On the host, open **Settings > Bluetooth & devices > Printers & scanners** and confirm the printer appears.
3. Print a test page from the host to confirm the driver and connection both work, not just that the queue was created.

## Troubleshoot

**The script succeeds but the printer doesn't appear**

`Add-Printer` returns success even when the connection URI is wrong, since it only creates the queue. Confirm the IP address, hostname, and resource path against the printer's own network settings page.

**"The specified driver is invalid" (vendor driver path only)**

The driver name in `-DriverName` doesn't match an installed driver exactly. Run `Get-PrinterDriver` on the host to see the exact name Windows registered, and use that string, not the name from the manufacturer's packaging.

See [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet) for the same task on other platforms.

<meta name="articleTitle" value="Deploy printers on Windows with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Deploy a driverless IPP printer to Windows hosts with a self-service script, or install a vendor driver first.">
