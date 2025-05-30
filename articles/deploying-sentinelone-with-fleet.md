# Deploying SentinelOne with Fleet

![Fleet and SentinelOne](../website/assets/images/articles/installing-sentinel-one-with-fleet-1600x900.png)

SentinelOne is a cybersecurity platform that provides endpoint protection, detection, and response capabilities to organizations. It uses artificial intelligence and machine learning to detect and prevent various types of cyber threats, including malware, ransomware, and zero-day exploits. It's a common toolset deployed by system admins through Fleet. This guide covers off deployment through macOS and Windows.

## MacOS

### Upload .mobileconfigs to Fleet

SentinelOne requires 5 separate mobileconfig files in order to properly function on macOS. Each of these serves an important operational function. These 5 profiles are available to download on my GitHub repo [here](https://github.com/harrisonravazzolo/Bluth-Company-GitOps/tree/main/lib/macos/SentinelOne). Let's quickly run through each one and highlight what it's actually doing on your endpoints.

> It's possible these profiles can be combined into one payload, but we've kept them separate here for troubleshooting purposes.

`s1_install_token.mobileconfig` - The simplest of the payloads. Find the key S1InstallRegistrationToken and replace the corresponding string value with your site token. This token can be found under the Sentinels tab for the corresponding site where you want to enroll your hosts.

`s1_network_extension.mobileconfig` - This payload allows SentinelOne's network monitoring system extension (com.sentinelone.network-monitoring) to be automatically loaded by macOS. It identifies the SentinelOne extension by its team identifier and makes the config mandatory by setting PayloadRemovalDisallowed to true.

`s1_network_filter.mobileconfig` - This configuration profile sets up the network filtering capabilities. It configures a web content filter that allows SentinelOne to monitor network traffic at the socket level (FilterSockets is true) while not filtering individual packets (FilterPackets is false). The profile ensures the network monitoring component is properly validated with Apple's security requirements and operates at the firewall grade level.

`s1_privacy_control.mobileconfig` - The privacy payload grants full disk access to three critical SentinelOne components: the main daemon (sentineld), the helper process (sentineld-helper), and the shell component (sentineld-shell). Additionally, it provides Bluetooth access permissions to the sentinel-helper component. All components are verified using Apple's code signing requirements with SentinelOne's team identifier.

`s1_system_extensions_disable.mobileconfig` - This profile prevents users from removing the network monitoring system extension. It designates the SentinelOne network monitoring extension (com.sentinelone.network-monitoring) as non-removable, prevents users from overriding this setting (AllowUserOverrides set to false) and identifies the legitimate extension using SentinelOne's team identifier. This profile complements the other SentinelOne configurations by ensuring users cannot disable or remove the network monitoring component through the macOS System Settings interface, maintaining continuous security protection on the device.

### Installer

From the SentinelOne admin console, navigate to the **Sentinels** tab on the left side pane and select **Packages**. Find the latest installer for macOS and your matching host architecture and click the icon to **Download**. 

From the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

>Working with different hardware architectures? Use labels to scope installs based on hardware.
 
On macOS, no pre-install or post-install script is required; however, the installer does support passing the site token as a flag if you prefer to deploy that route verses a configuration profile.

For admins that are leveraging the macOS Setup Experience in Fleet, we recommend adding the software to the list of items done on first boot.

## Windows

SentinelOne offers admins both an .exe and .msi installer, and Fleet recommends leveraging the .msi to deploy. These installers are better suited for enterprise environments with features like silent install and richer management capabilities at time of install. Additionally, the **Automatic install** functionality of Fleet is only available when deploying an .msi.

### Installer + script

After downloading the latest SentinelOne installer from your admin console, and retrieving your site token, from the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment. 

SentinelOne needs to be passed the site token at time of install, we can achieve this with an **Install Script**. Copy and paste this code snippet in Fleet and replace the variable with your unique value.

```
$logFile = "${env:TEMP}/fleet-install-software.log"
try {
    $installProcess = Start-Process msiexec.exe `
        -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`" SITE_TOKEN=YOUR_SITE_TOKEN_HERE" `
        -PassThru -Verb RunAs -Wait
    
    Get-Content $logFile -Tail 500
    
    # Convert exit code 3010 (restart required) to 0
    $exitCode = $installProcess.ExitCode
    if ($exitCode -eq 3010) {
        Write-Host "Installation successful but restart required, returning success code 0 to Fleet"
        Exit 0
    } else {
        Exit $exitCode
    }
} catch {
    Write-Host "Error: $_"
    Exit 1
}
```

Admin can add additional flags here, such as `/NORESTART`, check the SentinelOne documentation for a list of all flags that are supported.

## Linux

With support for both .rpm and .deb, deployment on Linux is straightforward. 

As in previous steps, find the latest installer for your Linux distro and **Download**.

From the **Software** tab in Fleet, **Add software** > **Custom package**. Upload the installer from the previous step. Select **Automatic install** or **Self-service** if those options apply to your environment.

### Post-install script

The default install script that is populated in Fleet is sufficient, but a post-install script is needed to set the site token and start the agent services. Here is an example post-install script that will set the token, start the service and check the status. Adjust the sleep time if needed.

```
#!/bin/bash

# Set the SentinelOne site token
sudo /opt/sentinelone/bin/sentinelctl management token set <YOUR_SITE_TOKEN_HERE>

# Start the SentinelOne service
sudo /opt/sentinelone/bin/sentinelctl control start

echo "Waiting 2 minutes for service to initialize..."
sleep 120

# Check the status of the SentinelOne service
sudo /opt/sentinelone/bin/sentinelctl control status
```

## Conclusion

Deploying SentinelOne through Fleet provides a streamlined approach to securing your endpoints across macOS, Windows, and Linux platforms. You can efficiently protect your organization from evolving cybersecurity threats with minimal deployment effort.

Want to learn more? Reach out directly to me or the [team at Fleet](https://fleetdm.com/contact) today!


<meta name="articleTitle" value="Deploying SentinelOne with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-04-15">
<meta name="description" value="Deploying SentinelOne with Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/installing-sentinel-one-with-fleet-1600x900.png">
