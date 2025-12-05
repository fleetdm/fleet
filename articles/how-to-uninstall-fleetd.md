# How to uninstall Fleet's agent (fleetd)

You can uninstall fleetd directly on a device or remotely through Fleet.


## Uninstall fleetd on macOS

To remove fleetd from a Mac:

1. Download the [macOS uninstall script](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh).
2. Open the **Terminal** app.
3. Navigate to where you saved the script: `cd /path/to/your/script`
4. Make the script executable: `chmod +x uninstall-fleetd-macos.sh`
5. Run the script: `sudo ./uninstall-fleetd-macos.sh`


## Uninstall fleetd on Windows

To remove fleetd from a Windows device:

1. Download the [Windows uninstall script](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/windows/scripts/uninstall-fleetd-windows.ps1).
2. Open **PowerShell** as administrator (right-click and select **Run as administrator**).
3. Navigate to where you saved the script: `cd C:\path\to\your\script`
4. Run the script: `.\uninstall-fleetd-windows.ps1`

> Note: When running unsigned PowerShell scripts, you are likely to receive a warning, and will need to adjust the [Execution Policy](https://learn.microsoft.com/en-gb/powershell/module/microsoft.powershell.security/set-executionpolicy?view=powershell-7.5). One example is: `Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process`. This will bypass all warnings and prompts for the current PowerShell session. 


## Uninstall fleetd on Linux

To remove fleetd from a Linux device:

1. Download the [Linux uninstall script](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/linux/scripts/uninstall-fleetd-linux.sh).
2. Open your terminal.
3. Navigate to where you saved the script: `cd /path/to/your/script`
4. Make the script executable: `chmod +x uninstall-fleetd-linux.sh`
5. Run the script: `sudo ./uninstall-fleetd-linux.sh`


## Uninstall fleetd remotely

To remove fleetd from a device through Fleet:

1. Add the uninstall script for [macOS](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh), [Windows](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/windows/scripts/uninstall-fleetd-windows.ps1), or [Linux](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/linux/scripts/uninstall-fleetd-linux.sh) to Fleet as a script.
2. Go to the device's **Host details** page.
3. Select **Actions > Run script** and choose the uninstall script.

After uninstalling, the device will show as offline in Fleet until you delete it.

Need help? Contact us through one of our [support channels](https://fleetdm.com/support).

<meta name="category" value="guides">
<meta name="authorFullName" value="Eric Shaw">
<meta name="authorGitHubUsername" value="eashaw">
<meta name="publishedOn" value="2021-09-08">
<meta name="articleTitle" value="How to uninstall fleetd">
<meta name="articleImageUrl" value="../website/assets/images/articles/how-to-uninstall-osquery-cover-1600x900@2x.jpg">
