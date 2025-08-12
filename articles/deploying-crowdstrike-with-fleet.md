# Deploy CrowdStrike with Fleet

CrowdStrike's Falcon platform is widely deployed by IT admins and security teams through centralized management consoles and automated deployment tools. A pillar in the security sector, it provides endpoint detection and response capabilities to organizations of all sizes. It uses artificial intelligence, machine learning, and behavioral analytics to detect and prevent sophisticated cyber threats, including advanced persistent threats (APTs), malware, ransomware, and zero-day exploits. This guide covers deployment and configuration across macOS, Windows, and Linux using Fleet.

There are multiple ways to install Crowdstrike, such as using the API method combined with scripts, listed [here.](https://github.com/CrowdStrike/falcon-scripts) Use whichever method is best for your organization.

## MacOS

### Upload .mobileconfigs to Fleet

CrowdStrike requires [5 separate mobileconfig files](https://github.com/fleetdm/fleet/tree/main/assets/configuration-profiles) in order to properly function on macOS.

> It's possible these profiles can be combined into one payload, but we've kept them separate here for troubleshooting purposes.

`crowdstrike-service-management.mobileconfig` - This payload is used to configure managed login items. A login item is an applications or service that automatically launches when a user logs in.

`crowdstrike-notification.mobileconfig` - It's often easiest for an admin to control the notifications and various banners that an application presents, reducing end-user interaction. This profile helps suppress items such as `ShowInLockScreen`.

`crowdstrike-system-extension` - An improvement on the classic macOS kernel extensions, or kexts, this validates the CrowdStrike extension in addition to preventing tampering and modification by your end users. The profile complements the other CrowdStrike configurations by ensuring users cannot disable or remove the network monitoring component through the macOS System Settings interface, maintaining continuous security protection on the device.

`crowdstrike-web-filter.mobileconfig` - This configuration profile configures the web filtering capabilities. It allows CrowdStrike to monitor network traffic at the socket level (FilterSockets is true) while not filtering individual packets (FilterPackets is false). A key component to comprehensive device protection, the filter component is properly validated with Apple's security requirements and operates at the firewall level.

`crowdstrike-full-disk-access.mobileconfig` - The privacy payload grants full disk access to CrowdStrike's application components. All components are verified using Apple's code signing requirements with CrowdStrike's team identifier.

### Installer

From your Falcon console, click **Host setup and management** > **Sensor Downloads**. Click the **Download** for the appropriate OS and architecture.

From the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

>Working with different hardware architectures? Use labels to scope installs based on hardware.

### Post-Install Script

The **Customer ID** used to assign hosts to your tenant and validate the license is passed through a script via `falconctl`. On the Sensor download page you will also find your Customer ID. In Fleet, define the following post-install script for the installer. 

```
#!/bin/bash

CUSTOMER_ID="YOUR-CUSTOMER-ID-HERE"
FALCON_PATH="/Applications/Falcon.app/Contents/Resources/falconctl"

sudo "$FALCON_PATH" license "$CUSTOMER_ID"

# Check status
if [ $? -eq 0 ]; then
    echo "Activation completed"
else
    echo "Activation failed"
    exit 1
fi
```

>If your org is using GitOps and want to pass the site key as a secret, follow this guide: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles

For admins that are leveraging the macOS Setup Experience in Fleet, we recommend adding the software to the list of items done on first boot.

## Windows

CrowdStrike offers admins both an .exe and .msi installer, and Fleet recommends leveraging the .msi to deploy. These installers are better suited for enterprise environments with features like silent install and richer management capabilities at time of install. Additionally, the **Automatic install** functionality of Fleet is only available when deploying an .msi.

### Installer + script

After downloading the latest CrowdStrike installer from your admin console, and retrieving your Customer ID, from the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

Falcon needs to be passed the Customer ID at time of install, we can achieve this with an **Install Script**. Copy and paste this code snippet in Fleet and replace the variable with your unique value.

```
# Set your Customer ID here
$FalconCid = "YOUR-CUSTOMER-ID-HERE"

$logFile = "${env:TEMP}/fleet-install-software.log"
try {
$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`" CID=${FalconCid}" `
  -PassThru -Verb RunAs -Wait
Get-Content $logFile -Tail 500
Exit $installProcess.ExitCode
} catch {
  Write-Host "Error: $_"
  Exit 1
}
```

There are several other flags that can be added, check the documentation for a list of supported options and their functionality.

## Linux

As in previous steps, find the latest installer for your Linux distro and **Download**.

From the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer. Select **Automatic install** or **Self-service** if those options apply to your environment.

### Post-install script

The default install script that is populated in Fleet is sufficient, but a post-install script is needed to set the Customer ID. Modify and add flags as needed for your deployment.

```
#!/bin/bash

CUSTOMER_ID="YOUR-CUSTOMER-ID-HERE"
FALCON_PATH="/opt/CrowdStrike/falconctl"

sudo "$FALCON_PATH" -s -cid="$CUSTOMER_ID"

# Check status
if [ $? -eq 0 ]; then
    echo "Activation completed"
else
    echo "Activation failed"
    exit 1
fi
```

Admins can verify the sensor installation by running a command searching for the falcon sensor `sudo ps -e | grep falcon-sensor`

## Conclusion

Fleet offers admins a simple approach to deploying the CrowdStrike Falcon sensor across the major operating systems. The lightweight Falcon sensor requires no restarts and offers a simple single-command installation, so you can efficiently protect your organization from evolving cybersecurity threats with minimal lift.

Want to learn more? Reach out directly to me or the [team at Fleet](https://fleetdm.com/contact) today!


<meta name="articleTitle" value="Deploy CrowdStrike with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-08-11">
<meta name="description" value="Deploy CrowdStrike with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png">
