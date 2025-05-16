# How to uninstall Fleet's agent (fleetd)

How to remove fleetd from your device:

1. Download the uninstall script for [macOS](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh), [Windows](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/windows/scripts/uninstall-fleetd-windows.ps1), or [Linux](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/linux/scripts/uninstall-fleetd-linux.sh).

2. Run the script.
- To run the script on macOS, open the **Terminal** app.
- Navigate to the script's directory by running this command: `cd /path/to/your/script`
- Make the script executable: `chmod +x uninstall-fleetd-macos.sh`
- Run it: `sudo ./uninstall-fleetd-macos.sh`

How to uninstall fleetd from a host via Fleet (remotely):

1. Add the uninstall script for [macOS](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh), [Windows](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/windows/scripts/uninstall-fleetd-windows.ps1), or [Linux](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/linux/scripts/uninstall-fleetd-linux.sh) hosts to Fleet.

2. Head to the host's **Host details** page and select **Actions > Run script** to run the script.

After performing these steps, the host will display as an offline host in the Fleet UI until you delete it.

Are you having trouble uninstalling Fleetd on macOS, Windows, or Linux? Get help [here](https://fleetdm.com/slack).

<meta name="category" value="guides">
<meta name="authorFullName" value="Eric Shaw">
<meta name="authorGitHubUsername" value="eashaw">
<meta name="publishedOn" value="2021-09-08">
<meta name="articleTitle" value="How to uninstall fleetd">
<meta name="articleImageUrl" value="../website/assets/images/articles/how-to-uninstall-osquery-cover-1600x900@2x.jpg">
