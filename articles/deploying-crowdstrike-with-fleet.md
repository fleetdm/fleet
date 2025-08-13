# Deploy CrowdStrike with Fleet

![Fleet and CrowdStrike](../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png)
This guide shows you how to deploy the CrowdStrike Falcon sensor on macOS, Windows, and Linux using Fleet. It covers uploading the required configuration profiles, installing the sensor, and passing your Customer ID for activation.

For reference, Crowdstrike Falcon install documentation can be found at:

https://github.com/CrowdStrike/falcon-scripts

## MacOS

### Upload .mobileconfigs to Fleet

CrowdStrike requires multiple `.mobileconfig` payloads on macOS. Each serves an important operational function.

> You can combine these into one payload, but we've kept them separate for troubleshooting purposes.

`crowdstrike-service-management.mobileconfig` - Configures managed login items so CrowdStrike services start automatically at login.

`crowdstrike-notification.mobileconfig` - Suppresses notifications and banners to reduce end-user interaction.

`crowdstrike-system-extension` - Approves the CrowdStrike system extension and prevents tampering through System Settings.

`crowdstrike-web-filter.mobileconfig` - Enables web filtering to monitor network traffic at the socket level.

`crowdstrike-full-disk-access.mobileconfig` - Grants full disk access to CrowdStrike components.

### Upload the installer

1. In the Falcon console, click **Host setup and management** > **Sensor Downloads**. 
2. Download the installer for the appropriate OS and architecture.
3. In Fleet, go to **Software > Add software > Custom package** to upload the installer
4. Select **Automatic install** or **Self-service** if those options apply to your environment.

 

>Use [labels](https://fleetdm.com/guides/managing-labels-in-fleet) to scope installs for different hardware architectures.

### Add a post-install script

The **Customer ID** is used to assign hosts to your tenant and validate the CrowdStrike Falcon license using a script that calls the `falconctl` binary.

> Your Customer ID can be found on the Sensor download page.

In Fleet, define the following post-install script for the CrowdStrike Falcon installer:

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

>For admins using the macOS Setup Experience in Fleet, we recommend adding the software to the list of items installed on first boot.

## Windows

CrowdStrike offers `.exe` and `.msi` Falcon installers for Windows. Using the `.msi` is preferred as this installer type performs a silent, fully-automated installation when using the **Automatic install** option in Fleet.

### Upload the installer

1. In the Falcon console, click **Host setup and management** > **Sensor Downloads**. 
2. Download the installer for the appropriate OS and architecture.
3. In Fleet, go to **Software > Add software > Custom package** to upload the installer
4. Select **Automatic install** or **Self-service** if those options apply to your environment.

> Use [labels](https://fleetdm.com/guides/managing-labels-in-fleet) to scope installs for different hardware architectures.

### Add a post install script

The **Customer ID** is used to assign hosts to your tenant and validate the CrowdStrike Falcon license using a script that calls the `falconctl` binary.

> Your Customer ID can be found on the Sensor download page.

In Fleet, define the following post-install script for the CrowdStrike Falcon installer, populating the value of the `$FalconCid` variable with your Customer ID string:

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

### Upload the installer

1. In the Falcon console, click **Host setup and management** > **Sensor Downloads**. 
2. Download the installer for the appropriate OS and architecture.
3. In Fleet, go to **Software > Add software > Custom package** to upload the installer
4. Select **Automatic install** or **Self-service** if those options apply to your environment.
> Use [labels](https://fleetdm.com/guides/managing-labels-in-fleet) to scope installs for different hardware architectures.```


### Add a post-install script

The **Customer ID** is used to assign hosts to your tenant and validate the CrowdStrike Falcon license using a script that calls the `falconctl` binary.
> Your Customer ID can be found on the Sensor download page.
In Fleet, define the following post-install script for the CrowdStrike Falcon installer, populating the value of the `$FalconCid` variable with your Customer ID string:

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



<meta name="articleTitle" value="Deploy CrowdStrike with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-08-11">
<meta name="description" value="Deploy CrowdStrike with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-crowdstrike-cover-800x450@2x.png">
