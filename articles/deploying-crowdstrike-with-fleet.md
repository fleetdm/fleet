# Deploy CrowdStrike with Fleet

CrowdStrike Falcon is a popular, highly-regarded cybersecurity platform. Falcon provides endpoint detection and response capabilities by using artificial intelligence, machine learning, and behavioral analytics to detect and prevent sophisticated attacks, including advanced persistent threats (APTs), malware, ransomware, and zero-day exploits. This guide covers CrowdStrike Falcon deployment and configuration across macOS, Windows, and Linux hosts using Fleet.

For reference, Crowdstrike Falcon install documentation can be found at:

https://github.com/CrowdStrike/falcon-scripts

## MacOS

### Upload .mobileconfigs to Fleet

CrowdStrike requires multiple `.mobileconfig` payloads on macOS. Each serves an important operational function.

> It's possible these profiles can be combined into one payload, but we've kept them separate here for troubleshooting purposes.

`crowdstrike-service-management.mobileconfig` - This payload is used to configure managed login items. A login item is an application or service that automatically launches when a user logs in.

`crowdstrike-notification.mobileconfig` - It's often easiest for an admin to control the notifications and various banners that an application presents to reduce end-user interaction and confusion. This profile helps supress items such as `ShowInLockScreen`.

`crowdstrike-system-extension` - An improvement on the classic macOS kernel extension, or kext, this validates the CrowdStrike Falcon extension in addition to preventing tampering and modification by your end users. The profile complements the other CrowdStrike configurations by ensuring users cannot disable or remove the network monitoring component through the macOS System Settings interface, maintaining continuous security protection on the device.

`crowdstrike-web-filter.mobileconfig` - This payload configures web filtering capabilities. It allows CrowdStrike to monitor network traffic at the socket level (FilterSockets is `true`) while not filtering individual packets (FilterPackets is `false`). A key component to comprehensive device protection, the filter is properly validated with Apple's security requirements and operates at the firewall level.

`crowdstrike-full-disk-access.mobileconfig` - This privacy preference payload grants full disk access to CrowdStrike Falcon. It uses the CrowdStrike Apple Developer team identifier to grant the Falcon application the necessary macOS entitlements for modifying Privacy controls.

### Installer

From your Falcon console, click **Host setup and management** > **Sensor Downloads**. Click **Download** for the appropriate OS and architecture.

From the **Software** tab in Fleet, select **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

>Working with different hardware architectures? Use labels to scope installs based on hardware.

### Post-Install Script

The **Customer ID** is used to assign hosts to your tenant and validate the CrowdStrike Falcon license using a script that calls the `falconctl` binary. Your Customer ID can be found on the Sensor download page. In Fleet, define the following post-install script for the CrwdStrike Falcon installer:

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

>If your org is using GitOps and you want to pass the site key as a secret, follow this guide: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles

>For admins that are leveraging the macOS Setup Experience in Fleet, we recommend adding the software to the list of items installed on first boot.

## Windows

CrowdStrike offers `.exe` and `.msi` Falcon installers for Windows. Using the `.msi` is preferred as this installer type performs a silent, fully-automated installation when using the **Automatic install** option in Fleet.

### Installer + script

After downloading the latest CrowdStrike Falcon `.msi` installer and retrieving your Customer ID from your admin console, navigate to the **Software** tab in Fleet, then select **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

The CrowdStrike Falcon tenant needs to collect the Customer ID from the host at time of install which we can be done with an **Install Script**. Copy and paste the code snippet below in Fleet and populate the value of the `$FalconCid` variable with your Customer ID string:

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

There are several other optional flags that can be added. Check the documentation for a list of supported options and their functionality.

## Linux

As in previous steps, find the latest installer for your Linux distro and **Download**.

From the **Software** tab in Fleet, select **Add software** > **Custom package**. Upload the installer. Select **Automatic install** or **Self-service** if those options apply to your environment.

### Post-install script

The default install script that is populated in Fleet is sufficient, but, a post-install script is needed to set the site token and start the agent services. Below is an example post-install script that will set the token, start the service and check the status. Adjust the sleep time if needed. Also, populate the value of the `$FalconCid` variable with your Customer ID string.

```
#!/bin/bash

# Set your Customer ID here
FalconCid = "YOUR-CUSTOMER-ID-HERE

echo "Setting CrowdStrike Falcon Customer ID: $FalconCid"

# Set the Customer ID
sudo /opt/CrowdStrike/falconctl -s --cid="$FalconCid"

# Check if the command was successful
if [ $? -eq 0 ]; then
    echo "Customer ID set successfully!"
    
    # Verify the setting
    echo "Verifying Customer ID..."
    sudo /opt/CrowdStrike/falconctl -g --cid
else
    echo "Error: Failed to set Customer ID"
    exit 1
fi
```

Admins can verify the installation by running a command searching for the falcon sensor with the following command:

`sudo ps -e | grep falcon-sensor`

## Conclusion

Fleet offers admins a straight-forward approach to deploying the CrowdStrike Falcon application across your macOS, Linux and Windows hosts.

Want to learn more or need additional information? Reach out directly to me or the [team at Fleet](https://fleetdm.com/contact) today!


<meta name="articleTitle" value="Deploy CrowdStrike with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-08-11">
<meta name="description" value="Deploy CrowdStrike with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png">
