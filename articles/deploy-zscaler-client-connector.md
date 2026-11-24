# Deploy Zscaler Client Connector with Fleet

Zscaler Client Connector isn't published to Homebrew or winget, so it isn't available as a Fleet-maintained app. Instead, add the installers from your Zscaler tenant to Fleet as custom packages, along with the configuration that lets them install without user prompts. When you're done, Zscaler Client Connector installs silently on macOS and Windows hosts, skips the enrollment page, and stays installed.

This guide covers the core install. Optional Zscaler features, like Endpoint DLP, anti-tampering, and SSL inspection certificates, need extra configuration profiles. Zscaler's documentation for each feature lists the settings they need.

## Prerequisites

- Fleet Premium.
- For macOS: [Apple MDM turned on](https://fleetdm.com/guides/apple-mdm-setup) in Fleet.
- A Zscaler Admin Console account with access to the Client Connector App Store.
- Your Zscaler cloud name (for example, `zscalertwo`) and the user domain from your identity provider's SAML NameID (for example, `example.com`).
- For GitOps: a [Fleet GitOps repo](https://fleetdm.com/docs/configuration/yaml-files) and a download URL for each installer that Fleet can reach without signing in.

## Download the installers

1. In the Zscaler Admin Console, go to **Infrastructure > Client Connector > App Store**.
2. On the **New Releases** tab, select **macOS** and download the `.pkg` file.
3. Select **Windows** and download the `.msi` file.

> **Note:** On the **New Releases** tab, make sure the build you downloaded is enabled. Zscaler only offers enabled builds when Client Connector updates itself, so a disabled build can lead to hosts running a different version than the one you deployed.

## Add the macOS configuration profiles

Add the profiles before you add the macOS package. If Zscaler Client Connector installs before the profiles arrive, users are asked to approve the system extension, and the app opens to its enrollment page instead of your sign-in page.

### Allow the system extension and login items

Save this as `zscaler-client-connector-extensions.mobileconfig`. It allows Zscaler's system extensions (Zscaler's team ID is `PCBCQZJ7S7`) and adds a managed login items rule so users can't turn off Zscaler's background items:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.apple.system-extension-policy</string>
      <key>PayloadIdentifier</key>
      <string>com.example.zscaler.system-extension-policy</string>
      <key>PayloadUUID</key>
      <string>6A1D3E62-8C0B-4B0E-9F27-3E5D1B0C7A41</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>AllowUserOverrides</key>
      <false/>
      <key>AllowedTeamIdentifiers</key>
      <array>
        <string>PCBCQZJ7S7</string>
      </array>
    </dict>
    <dict>
      <key>PayloadType</key>
      <string>com.apple.servicemanagement</string>
      <key>PayloadIdentifier</key>
      <string>com.example.zscaler.servicemanagement</string>
      <key>PayloadUUID</key>
      <string>0F4C8B1E-2D7A-4C55-8E3B-9A6F1D2C4B7E</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>Rules</key>
      <array>
        <dict>
          <key>RuleType</key>
          <string>TeamIdentifier</string>
          <key>RuleValue</key>
          <string>PCBCQZJ7S7</string>
          <key>Comment</key>
          <string>Zscaler</string>
        </dict>
      </array>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Zscaler Client Connector extensions</string>
  <key>PayloadIdentifier</key>
  <string>com.example.zscaler.extensions</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>B3E7A9D4-5C21-4F8E-A6B0-7D1E2F3C4A59</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

> **Note:** If you use Transparent Proxy-based Interception, add a separate profile with a VPN payload for Zscaler's transparent proxy extension. Zscaler specifies these settings: VPN type **Custom SSL**, identifier `com.zscaler.zscaler`, server `Zscaler server`, provider bundle identifier `com.zscaler.zscaler.TRPTunnel`, provider type **App-proxy**, and designated requirement `identifier "com.zscaler.zscaler.TRPTunnel" and anchor apple generic`. Also turn on the option that keeps users from disabling on-demand VPN.

### Set the install parameters

On macOS, Zscaler Client Connector reads its install parameters from the `com.zscaler.installparams` preference domain. Save this as `zscaler-client-connector-install-parameters.mobileconfig`, and replace `zscalertwo` and `example.com` with your cloud name and user domain:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.zscaler.installparams</string>
      <key>PayloadIdentifier</key>
      <string>com.example.zscaler.installparams</string>
      <key>PayloadUUID</key>
      <string>C8D2F6A1-3B4E-4D7C-9E05-1A2B3C4D5E6F</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>installation-parameters</key>
      <dict>
        <key>cloudName</key>
        <string>zscalertwo</string>
        <key>userDomain</key>
        <string>example.com</string>
        <key>launchTray</key>
        <string>1</string>
      </dict>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Zscaler Client Connector install parameters</string>
  <key>PayloadIdentifier</key>
  <string>com.example.zscaler.installparams.profile</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>D4E5F6A7-B8C9-4D0E-8F1A-2B3C4D5E6F70</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Zscaler expects every value as a string, including `1` and `0`. To add other options, like `policyToken` and `strictEnforcement`, see Zscaler's [supported parameters for macOS](https://help.zscaler.com/zscaler-client-connector/supported-parameters-zscaler-client-connector-macos).

> **Note:** To keep a policy token out of your repo, store it as a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) and set `policyToken` to `$FLEET_SECRET_ZSCALER_POLICY_TOKEN`. Fleet replaces the variable when it sends the profile to the host.

### Upload the profiles

1. In Fleet, go to **Controls > OS settings > Configuration profiles** and select your fleet.
2. Select **Add profile** and upload `zscaler-client-connector-extensions.mobileconfig`.
3. Repeat step 2 for `zscaler-client-connector-install-parameters.mobileconfig`.

## Add the macOS package

1. Go to **Software** and select your fleet.
2. Select **Add software > Custom package** and upload the `.pkg` file.
3. Turn on **Deploy** to install Zscaler Client Connector on every host in the fleet that doesn't have it. Fleet creates a policy that checks for the app and installs it on hosts that fail.
4. Select **Add software**.

The default install script works as is, since the install parameters come from the configuration profile.

> **Note:** The Zscaler `.pkg` also includes an uninstaller app. After you add the package, go to **Policies**, open the policy Fleet created, and confirm its query checks for `com.zscaler.zscaler`. If it checks for `com.zscaler.installer.uninstall`, change the bundle ID to `com.zscaler.zscaler`.

## Add the Windows package

On Windows, pass the install parameters as MSI properties in the install script.

1. Go to **Software** and select your fleet.
2. Select **Add software > Custom package** and upload the `.msi` file.
3. Turn on **Deploy**.
4. Select **Advanced options** and replace the install script with this one. Replace `zscalertwo` and `example.com` with your cloud name and user domain:

   ```powershell
   $logFile = "${env:TEMP}/fleet-install-software.log"

   # 3010 and 1641 mean the install succeeded but needs a reboot.
   $successCodes = @(0, 3010, 1641)

   try {

   $installProcess = Start-Process msiexec.exe `
     -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${env:INSTALLER_PATH}`" CLOUDNAME=zscalertwo USERDOMAIN=example.com" `
     -PassThru -Verb RunAs -Wait

   Get-Content $logFile -Tail 500

   if ($successCodes -contains $installProcess.ExitCode) {
     Exit 0
   }

   Exit $installProcess.ExitCode

   } catch {
     Write-Host "Error: $_"
     Exit 1
   }
   ```

5. Select **Add software**.

For more properties, like `POLICYTOKEN` and `STRICTENFORCEMENT`, see Zscaler's [supported parameters for Windows](https://help.zscaler.com/zscaler-client-connector/supported-parameters-zscaler-client-connector-windows). Use the MSI names, which are all caps.

> **Warning:** Custom variables (`$FLEET_SECRET_*`) aren't supported in install scripts for `.msi` packages. Any policy token you add here is stored in the script as plain text and visible to anyone who can view the software in Fleet.

> **Note:** Windows only reads MSI properties during install. To change them later, update the install script and reinstall Zscaler Client Connector.

## Deploy with GitOps

Skip this section if you manage Fleet in the UI.

With GitOps, you give Fleet your organization's Zscaler download URL once. Fleet downloads the installer from that URL, adds it as a package, and installs it on hosts that don't have it. Each time GitOps runs, Fleet checks the URL again and downloads the installer only if the file changed.

> **Note:** Fleet downloads the installer without signing in, so the URL must be publicly reachable. If your Zscaler download link requires a sign-in, download the installers yourself and host them somewhere Fleet can reach, like an S3 bucket. Or add them in the Fleet UI first and reference them by `hash_sha256` instead of `url`.

1. Save the two `.mobileconfig` files from [Add the macOS configuration profiles](#add-the-macos-configuration-profiles) in your repo, for example in `lib/macos/configuration-profiles/`.
2. Save the Windows install script from [Add the Windows package](#add-the-windows-package) as `lib/windows/scripts/zscaler-client-connector-install.ps1`.
3. Create `lib/macos/software/zscaler-client-connector.package.yml`:

   ```yaml
   - url: https://example.com/installers/Zscaler-osx-installer.pkg
   ```

   Replace the URL with your organization's macOS download URL.

4. Create `lib/windows/software/zscaler-client-connector.package.yml`:

   ```yaml
   - url: https://example.com/installers/Zscaler-windows-installer.msi
     install_script:
       path: ../scripts/zscaler-client-connector-install.ps1
   ```

   Replace the URL with your organization's Windows download URL.

5. In your fleet's YAML file (for example, `fleets/workstations.yml`), add the profiles, packages, and a policy for each platform that installs Zscaler Client Connector when it's missing:

   ```yaml
   controls:
     apple_settings:
       configuration_profiles:
         - path: ../lib/macos/configuration-profiles/zscaler-client-connector-extensions.mobileconfig
         - path: ../lib/macos/configuration-profiles/zscaler-client-connector-install-parameters.mobileconfig

   software:
     packages:
       - path: ../lib/macos/software/zscaler-client-connector.package.yml
       - path: ../lib/windows/software/zscaler-client-connector.package.yml

   policies:
     - name: macOS - Zscaler Client Connector installed
       query: SELECT 1 FROM apps WHERE bundle_identifier = 'com.zscaler.zscaler';
       platform: darwin
       install_software:
         package_path: ../lib/macos/software/zscaler-client-connector.package.yml
     - name: Windows - Zscaler Client Connector installed
       query: SELECT 1 FROM programs WHERE name = 'Zscaler' AND publisher = 'Zscaler Inc.';
       platform: windows
       install_software:
         package_path: ../lib/windows/software/zscaler-client-connector.package.yml
   ```

6. Merge the change.

On macOS, Zscaler Client Connector installs to `/Applications/Zscaler/Zscaler.app` with the bundle ID `com.zscaler.zscaler`. On Windows, it shows up in installed programs as `Zscaler`, published by `Zscaler Inc.`.

## Handle updates

Zscaler Client Connector can update itself. In the Zscaler Admin Console, go to **Infrastructure > Client Connector > App Store** and configure the update settings to choose which version hosts update to and to roll out new versions in phases. Learn more in [Zscaler's App Store docs](https://help.zscaler.com/zscaler-client-connector/about-zscaler-client-connector-app-store).

Letting Zscaler handle updates works well alongside Fleet. The install policies only check that the app is installed, not which version, so Fleet won't reinstall an older version over a newer one.

Fleet still installs its own copy of the installer on new hosts, so keep that copy current:

- **GitOps:** If your download URL always serves the latest installer, you don't need to change anything. Fleet picks up a new installer the next time GitOps runs. To pick up new versions without waiting for a change to your repo, run your GitOps workflow on a schedule. If the URL points to a specific version, update the `url` in the package YAML file and merge the change.
- **Fleet UI:** On the software's details page, select **Edit** next to the package and upload the new installer.

## Verify

1. In Fleet, go to **Hosts** and select a host in the fleet.
2. On the **Software** tab, confirm Zscaler Client Connector is installed.
3. On macOS hosts, select **OS settings** and confirm both Zscaler profiles show as **Verified**.
4. On the host, open Zscaler Client Connector. It opens to your identity provider's sign-in page, not the Zscaler enrollment page that asks for a domain.
5. On macOS, open **System Settings > General > Login Items & Extensions**. Zscaler's extensions are allowed, and the user wasn't asked to approve them.

## Troubleshoot

**macOS asks the user to allow a system extension**

The extensions profile didn't reach the host before the install. Check that the profile shows as **Verified** on the host's **OS settings**. The system extension is approved once the profile installs, so you don't need to reinstall the app.

**The app asks the user for a domain or cloud**

The install parameters didn't reach Zscaler Client Connector. On macOS, check that the install parameters profile is **Verified**, then quit and reopen the app. On Windows, check the install script for typos in `CLOUDNAME` and `USERDOMAIN`, then reinstall.

**GitOps fails to add the package**

Fleet couldn't download the installer from `url`. Confirm the link works in a private browser window without signing in. If it doesn't, host the installer somewhere public or switch to `hash_sha256`.

## Further reading

- [Deploy software](https://fleetdm.com/guides/deploy-software-packages)
- [Automatically install software](https://fleetdm.com/guides/automatic-software-install-in-fleet)
- [Custom OS settings](https://fleetdm.com/guides/custom-os-settings)
- [Zscaler Client Connector: supported parameters for macOS](https://help.zscaler.com/zscaler-client-connector/supported-parameters-zscaler-client-connector-macos)
- [Zscaler Client Connector: supported parameters for Windows](https://help.zscaler.com/zscaler-client-connector/supported-parameters-zscaler-client-connector-windows)
- [Zscaler Client Connector: install options for MSI](https://help.zscaler.com/zscaler-client-connector/customizing-zscaler-client-connector-install-options-msi)

<meta name="articleTitle" value="Deploy Zscaler Client Connector with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-20">
<meta name="category" value="guides">
<meta name="description" value="Install Zscaler Client Connector on macOS and Windows with Fleet, including install parameters and the profiles macOS needs.">
