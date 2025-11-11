# Deploy CrowdStrike Falcon with Fleet

This guide will cover how to deploy CrowdStrike Falcon on macOS, Linux and Windows using Fleet. It includes:

- Installing the CrowdStrike Falcon application
- Creating a post-install script to collect the CrowdStrike Customer ID for activation
- Deploying required application configurations

### Install notes

- Fleet recommends using the End User Setup Experience to install CrowdStrike on hosts when they are initially enrolled and provisioned.
  - [macOS Setup](https://fleetdm.com/guides/macos-setup-experience#install-software)
  - [Linux](https://fleetdm.com/guides/windows-linux-setup-experience#choose-software)
  - [Windows](https://fleetdm.com/guides/windows-linux-setup-experience#choose-software)
 
- If your organization is using Fleet GitOps and you want to pass the CrowdStrike site key as a secret, follow this guide: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles.

- Starting with fleetd version 1.50, you can use the `crowdstrike_falcon` osquery table to check the status of a Crowdstrike Falcon installation on macOS and Linux.

## Download the Falcon installer

On the CrowdStrike Falcon dashboard, click the hamburger menu in the top-left corner of the page, then navigate to **Host setup and management** > **Sensor Downloads** (in the **Deploy** section of the menu).

Select the appropriate Falcon Sensor package for your platform and copy the **Customer ID** string found in the **How to install** sidebar on the right side of the page. The **Customer ID** *must be collected* during the CrowdStrike installation to activate the Falcon application on a host.

> For Windows, CrowdStrike offers `.exe` and `.msi` Falcon installers. Selecting the `.msi` package is preferable because it performs a silent, fully-automated installation when using the **Automatic install** option in Fleet.

See the sections below for more steps specific to your platform.

## macOS Falcon installation

### 1. Deploy configuration profiles

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

### 2. Create a post-install script

To activate a host in the CrowdStrike tenant, a script must be excuted after CrowdStrike Falcon is installed on the host to collect the **Customer ID**. Use this script on macOS with the **Customer ID** string copied from your CrowdStrike tenant above:

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

## Linux Falcon installation

### 1. Create a post-install script

To activate a host in the CrowdStrike tenant, a script must be excuted after CrowdStrike Falcon is installed on the host to collect the **Customer ID**. Use this script on Linux with the **Customer ID** string copied from your CrowdStrike tenant above:

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

## Windows Falcon installation

### 1. Create a post-install script

To activate a host in the CrowdStrike tenant, a script must be excuted after CrowdStrike Falcon is installed on the host to collect the **Customer ID**. Use this script on Windows with the **Customer ID** string copied from your CrowdStrike tenant above:

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

## Conclusion

Fleet offers admins a straight-forward approach to deploying the CrowdStrike Falcon application across your macOS, Linux and Windows hosts. See https://fleetdm.com/guides/deploy-software-packages for more information on installing software packages using Fleet.

<meta name="articleTitle" value="Deploy CrowdStrike with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-11-05">
<meta name="description" value="Deploy CrowdStrike with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png">
