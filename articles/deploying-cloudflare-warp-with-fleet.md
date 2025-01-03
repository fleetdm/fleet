# Deploying Cloudflare WARP with Fleet

Cloudflare WARP is a secure VPN-like service that encrypts internet traffic and routes it through Cloudflare's network, improving privacy and security without compromising speed.

## MacOS

1. Create custom MDM Config
   a. Download the example `.mobileconfig` file
   b. Tailor the payload with the [desired parameters](https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/deployment/mdm-deployment/parameters/) to satisfy your deployment

2. Upload `.mobileconfig` to Fleet
   a. In the Fleet admin console, navigate to **Controls**
   b. Select the **Team** that requires Cloudflare WARP
   c. Select **OS settings** > **Custom settings**
   d. Select **Add profile** and upload the `.mobileconfig` from step 1
   e. Select the hosts which require Cloudflare WARP:
      - **All hosts:** Deploys WARP to all hosts in selected Team
      - **Custom:** Deploys WARP to a subset of the hosts in the Team using [labels](https://fleetdm.com/guides/managing-labels-in-fleet)

> Note that the payload will be installed on all targeted hosts, but the WARP agent is not yet installed. Proceed to step 3 to complete the process.

3. Install WARP on hosts
   a. In the Fleet admin console, navigate to **Software**
   b. Select the **Team** that requires Cloudflare WARP
   c. Select **Add software**
      - Either add Cloudflare WARP from the **Fleet-maintained** library or
      - Upload a custom `.pkg` obtained from [Cloudflare.](https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/download-warp/#macos) If deploying with this approach, WARP will still need to be installed on select hosts via the UI, API or GitOps. Learn more about deploying software from this [article.](https://fleetdm.com/guides/deploy-software-packages)

> If using Fleet-maintained app, you can choose to install on hosts automatically or manually. To allow users to install WARP from Fleet Desktop, check the box for Self-service.

## Windows

1. Download the WARP installer for Windows
   a. Visit the [Download](https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/download-warp/#windows) page to review system requirements and download the installer for your OS.

2. Upload WARP installer to Fleet
   a. In the Fleet admin console, navigate to **Software**
   b. Select the **Team** that requires Cloudflare WARP
   c. Select **Add software** > **Custom Package** and upload the `.msi` file downloaded from step 1
      - To allow users to install WARP from Fleet Desktop, select Self-service. (Optional)
   d. Select **Advanced options**
   e. In **Install script**, replace the default script:

   ```
   $logFile = "${env:TEMP}/fleet-install-software.log"

   try {

   $installProcess = Start-Process msiexec.exe `
   -ArgumentList "/quiet /norestart ORGANIZATION=your-team-name SUPPORT_URL=https://example.com /lv ${logFile} /i `"${env:INSTALLER_PATH}`"" `
   -PassThru -Verb RunAs -Wait

   Get-Content $logFile -Tail 500

   Exit $installProcess.ExitCode

   } catch {
   Write-Host "Error: $_"
   Exit 1
   }
   ```

> Refer to Cloudflare's [deployment parameters](https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/deployment/mdm-deployment/parameters/) for a description of each argument and adjust your script as needed.

4. Install WARP on hosts
   a. In the Fleet admin console, navigate to **Hosts**
   b. Select the host that requires the WARP client
   c. Go to **Software** and search for **Cloudflare WARP**
   d. Select **Actions** > **Install**
   
> Learn more about ways to deploy software via the UI, API or GitOps from this [article.](https://fleetdm.com/guides/deploy-software-packages)

## Linux

Fleet allows admins to execute custom scripts on Linux hosts. The following example script creates an [MDM file](https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/deployment/mdm-deployment/#linux) and installs WARP on an Ubuntu host:

```
#!/bin/sh

# Write the mdm.xml file
touch /var/lib/cloudflare-warp/mdm.xml
echo -e "<dict>\n   <key>organization</key>\n   <string>your-team-name</string>\n</dict>
" > /var/lib/cloudflare-warp/mdm.xml

# Add cloudflare gpg key
curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | sudo gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg

# Add this repo to your apt repositories
echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list

# Install
sudo apt-get -y update && sudo apt-get -y install cloudflare-warp
```

> To learn about deploying scripts across multiple hosts, check out this [article.](https://fleetdm.com/guides/policy-automation-run-script)

To install WARP on other Linux distributions, refer to the [package repository](https://pkg.cloudflareclient.com/)

<meta name="articleTitle" value="Deploying Cloudflare WARP with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-12-20">
