# Deploy CrowdStrike Falcon with Fleet

This guide will show you how to deploy CrowdStrike Falcon on macOS, Linux and Windows using Fleet. It covers installing the CrowdStrike Falcon application, creating a post-install script for collecting the CrowdStrike Customer ID for activation and deploying required application configurations.

You can use Setup Experience to install CrowdStrike on [macOS](https://fleetdm.com/guides/macos-setup-experience#install-software), [Windows](https://fleetdm.com/guides/windows-linux-setup-experience#choose-software), and [Linux](https://fleetdm.com/guides/windows-linux-setup-experience#choose-software) hosts when they are initially provisioned.

> Starting with fleetd 1.50, you can use the `crowdstrike_falcon` osquery table to check the status of a Crowdstrike Falcon installation on macOS and Linux.

## Get the Falcon installer

From the CrowdStrike Falcon dashboard, click the hamburger menu in the top-left corner of the page, then navigate to **Host setup and management** > **Sensor Downloads** (in the **Deploy** section of the menu).

Once you select the appropriate Falcon Sensor package for your platform, make note of your **Customer ID**, found in the **How to install** sidebar on the right side of the page. You'll need this below.

> For Windows, CrowdStrike offers `.exe` and `.msi` Falcon installers. The `.msi` installer performs a silent, fully-automated installation when using the **Automatic install** option in Fleet, so you'll likely want that one.

## macOS

### 1. Set up configuration profiles

CrowdStrike Falcon requires multiple `.mobileconfig` payloads on macOS.

The payloads can be combined and delivered as a single Configuration Profile, or, delivered in separate Configuration Profiles for modularity and easier reading.

Below is an explanation of what each of the macOS CrowdStrike Falcon payloads does:

- `crowdstrike-service-management.mobileconfig` - Configure CrowdStrike Falcon as a managed login item so its services can't be stopped by end users.
- `crowdstrike-notification.mobileconfig` - Suppress notifications to reduce end user notification fatigue. (This is a best practice for many fully-managed applications.)
- `crowdstrike-system-extension` - Install the CrowdStrike Falcon System Extension to allow all necessary application entitlements and access to the macOS kernel.
- `crowdstrike-web-filter.mobileconfig` - Enable web filtering to monitor network traffic at the socket level.
- `crowdstrike-full-disk-access.mobileconfig` - Grant full disk access to all CrowdStrike application processes using the CrowdStrike Apple Developer team identifier.

[Download the CrowdStrike Falcon macOS Configuration Profiles](https://github.com/fleetdm/fleet/tree/main/docs/solutions/macos/configuration-profiles)

To upload Configuration Profiles to your Fleet instance: go to **Controls > OS Settings > Custom settings** then click **Add Profile**.

![Manage configuration profiles](../website/assets/images/articles/fleet-crowdstrike-add-profile-800x450@2x.png)

### 2. Prepare the post-install script

To match a host to your CrowdStrike account, you'll need to run a script after Falcon is installed. You can use the script below for macOS, combined with the Customer ID you grabbed earlier.

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

### 3. Add the Falcon Sensor to your software library

1. In Fleet, go to **Software > Add software > Custom package** to upload the Falcon Sensor installer.
2. Click **Advanced options**, then paste the activation script from the previous step into **Post-install script**, making sure to set the `CUSTOMER_ID` variable.

![Add software advanced options](../website/assets/images/articles/fleet-crowdstrike-post-install-script-800x450@2x.png)

3. Click **Add software**.

For more information on adding software, see the [software deployment guide](https://fleetdm.com/guides/deploy-software-packages).

## Linux

### 1. Prepare the post-install script

To match a host to your CrowdStrike account, you'll need to run a script after Falcon is installed. You can use the script below for Linux, combined with the Customer ID you grabbed earlier.

```
#!/bin/bash
CUSTOMER_ID="YOUR-CUSTOMER-ID-HERE"

# Set the Customer ID
sudo /opt/CrowdStrike/falconctl -s --cid="$CUSTOMER_ID"

if [ $? -eq 0 ]; then
    echo "Activation completed"
else
    echo "Activation failed"
    exit 1
fi
```

CrowdStrike provides [documentation for additional flags](https://github.com/CrowdStrike/falcon-scripts/tree/main/bash/install) you can use here.

### 2. Add the Falcon Sensor to your software library

1. In Fleet, go to **Software > Add software > Custom package** to upload the Falcon Sensor installer.
2. Click **Advanced options**, then paste the activation script from the previous step into **Post-install script**, making sure to set the `CUSTOMER_ID` variable.

> You use [labels](https://fleetdm.com/guides/managing-labels-in-fleet) to scope installations for different hardware architectures.

3. Click **Add software**.

For more information on adding software, see the [software deployment guide](https://fleetdm.com/guides/deploy-software-packages).

## Windows

### 1. Prepare the post-install script

To match a host to your CrowdStrike account, you'll need to run a script after Falcon is installed. You can use the script below for Windows, combined with the Customer ID you grabbed earlier.

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

CrowdStrike provides [documentation for additional flags](https://github.com/CrowdStrike/falcon-scripts/tree/main/powershell/install) you can use here.

### 2. Add the Falcon Sensor to your software library

1. In Fleet, go to **Software > Add software > Custom package** to upload the Falcon Sensor installer.
2. Click **Advanced options**, then paste the activation script from the previous step into **Post-install script**, making sure to set the `$FalconCid` variable.
3. Click **Add software**.

For more information on adding software, see the [software deployment guide](https://fleetdm.com/guides/deploy-software-packages).

<meta name="articleTitle" value="Deploy CrowdStrike with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-11-05">
<meta name="description" value="Deploy CrowdStrike with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png">
