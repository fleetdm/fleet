# Deploying custom osquery extensions in Fleet: A step-by-step guide

### Links to article series:

- Part 1: [Deploying custom osquery extensions in Fleet](https://fleetdm.com/articles/deploying-custom-osquery-extensions-in-fleet)
- Part 2: Deploying custom osquery extensions in Fleet: A step-by-step guide

Fleet collects data from your hosts using osquery, an open-source agent that lets you query a computer's state (installed software, running processes, and so on) using SQL. An osquery extension is a small program that adds new tables you can query, for data osquery doesn't collect out of the box. This guide walks you through packaging a custom extension, testing it, and deploying it to your hosts as a single install, on macOS, Windows, and Linux.

## Prerequisites

- A working, compiled osquery extension binary for your target platform. Name it with a `.ext` extension on macOS and Linux, and `.ext.exe` on Windows.
- Admin access to a fleet in Fleet, with [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).
- Platform-specific packaging tools: Xcode Command Line Tools (macOS), the [WiX Toolset](https://wixtoolset.org/) v3 (Windows), and `dpkg-deb` (Linux, included on Debian and Ubuntu).

> If you're deploying custom extensions across many hosts through a self-hosted TUF server instead, see the [agent configuration guide](https://fleetdm.com/docs/configuration/agent-configuration#extensions) for that setup. This guide covers the simpler, direct deployment approach: no TUF server required.

## Step 1: Package the extension binary

Whether you wrote the extension yourself, had an AI assistant write it, or downloaded it from the internet, you need to wrap the binary in an installer before Fleet can deploy it.

### macOS

1. Create a folder that mirrors where the extension should live on disk, and copy the binary into it:

   ```
   mkdir -p payload/usr/local/bin
   cp my-custom-extension.ext payload/usr/local/bin/
   ```

2. Set the ownership and permissions the extension needs to run. Fleet requires the extension file be owned by `root:wheel`, with execute permissions (typically `755`):

   ```
   chmod 755 payload/usr/local/bin/my-custom-extension.ext
   sudo chown root:wheel payload/usr/local/bin/my-custom-extension.ext
   ```

3. Build the package with `pkgbuild`:

   ```
   pkgbuild \
     --root payload \
     --identifier com.yourcompany.my-custom-extension \
     --version 1.0 \
     --install-location / \
     my-custom-extension.pkg
   ```

   `pkgbuild` preserves the ownership and permissions from your `payload` folder, so the extension lands on the host ready to run.

### Windows

1. Create a `.wxs` file describing the install, for example `my-custom-extension.wxs`:

   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
     <Product Id="*" Name="My custom extension" Language="1033" Version="1.0.0.0"
              Manufacturer="Your company" UpgradeCode="PUT-A-GUID-HERE">
       <Package InstallerVersion="200" Compressed="yes" InstallScope="perMachine" />
       <MediaTemplate EmbedCab="yes" />
       <Directory Id="TARGETDIR" Name="SourceDir">
         <Directory Id="ProgramFilesFolder">
           <Directory Id="INSTALLFOLDER" Name="MyCustomExtension" />
         </Directory>
       </Directory>
       <DirectoryRef Id="INSTALLFOLDER">
         <Component Id="ExtensionBinary" Guid="PUT-A-GUID-HERE">
           <File Id="ExtensionExe" Source="my-custom-extension.ext.exe" KeyPath="yes" />
         </Component>
       </DirectoryRef>
       <Feature Id="MainFeature" Title="Main" Level="1">
         <ComponentRef Id="ExtensionBinary" />
       </Feature>
     </Product>
   </Wix>
   ```

   Generate the two GUIDs with `[guid]::NewGuid()` in PowerShell.

2. Compile and link the package with WiX:

   ```
   candle my-custom-extension.wxs
   light my-custom-extension.wixobj -o my-custom-extension.msi
   ```

### Linux

1. Create the package's directory layout and copy the binary into it:

   ```
   mkdir -p my-custom-extension_1.0_amd64/usr/local/bin
   mkdir -p my-custom-extension_1.0_amd64/DEBIAN
   cp my-custom-extension.ext my-custom-extension_1.0_amd64/usr/local/bin/
   chmod 755 my-custom-extension_1.0_amd64/usr/local/bin/my-custom-extension.ext
   ```

2. Add a `DEBIAN/control` file with the package metadata:

   ```
   Package: my-custom-extension
   Version: 1.0
   Section: utils
   Priority: optional
   Architecture: amd64
   Maintainer: you@yourcompany.com
   Description: Custom osquery extension
   ```

3. Build the package:

   ```
   dpkg-deb --build --root-owner-group my-custom-extension_1.0_amd64
   ```

   For an `.rpm` instead, use `rpmbuild` with an equivalent spec file.

## Step 2: Test the extension locally

Test the extension before deploying it through Fleet.

1. Run the extension against a local osquery shell:

   ```
   $ sudo /path/to/orbit shell -- --extension /path/to/my-custom-extension.ext
   ```

2. At the `osquery>` prompt, confirm the extension loaded and query one of its tables:

   ```
   osquery> SELECT * FROM osquery_extensions;
   osquery> SELECT * FROM your_custom_table;
   ```

   If the extension doesn't appear in `osquery_extensions`, check the osquery log (`/var/log/osquery/osqueryd.results.log` on macOS and Linux, `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` on Windows) for the load error.

3. Once the extension behaves as expected locally, install the package you built in step 1 on a single test host and confirm it loads there too, before rolling it out to the rest of your fleet.

## Step 3: Add the post-install script

Configuring the extension loader and restarting Orbit are normally two separate manual steps after installing the binary. Fold both into the package's post-install script so deploying the package through Fleet is the only thing an admin has to do.

Orbit (Fleet's agent) loads extensions from a file named `extensions.load` in its root directory: `/opt/orbit/` on macOS and Linux, `C:\Program Files\Orbit\` on Windows. This file lists the full path to each extension binary, one per line.

When you upload the package on the [Software page](https://fleetdm.com/guides/deploy-software-packages), open **Advanced options** and set the post-install script for your platform.

### macOS

```bash
#!/bin/bash

set -e

EXTENSION_PATH="/usr/local/bin/my-custom-extension.ext"
ORBIT_ROOT="/opt/orbit"
EXTENSIONS_LOAD_FILE="$ORBIT_ROOT/extensions.load"

if ! grep -qxF "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE" 2>/dev/null; then
  echo "$EXTENSION_PATH" >> "$EXTENSIONS_LOAD_FILE"
fi

# Restart Orbit in a detached process, 10 seconds after this script exits.
# Restarting Orbit immediately would kill this script before it finishes,
# since Orbit is the process running it.
(sleep 10 && launchctl kickstart -k system/com.fleetdm.orbit) >/dev/null 2>&1 &
disown
```

### Windows

```powershell
$ExtensionPath = "C:\Program Files\MyCompany\MyCustomExtension\my-custom-extension.ext.exe"
$OrbitRoot = "C:\Program Files\Orbit"
$ExtensionsLoadFile = Join-Path $OrbitRoot "extensions.load"

if (-not (Test-Path $ExtensionsLoadFile) -or -not (Select-String -Path $ExtensionsLoadFile -Pattern ([regex]::Escape($ExtensionPath)) -Quiet)) {
    Add-Content -Path $ExtensionsLoadFile -Value $ExtensionPath
}

# Restart the Orbit service in a detached process, 10 seconds after this script
# exits. Restarting it immediately would kill this script before it finishes,
# since the Orbit service is running it.
Start-Process powershell.exe -WindowStyle Hidden -ArgumentList `
    "-NoProfile -Command Start-Sleep -Seconds 10; Restart-Service -Name 'Fleet osquery' -Force"
```

### Linux

Same as the macOS script, with the last two lines replaced to restart Orbit's `systemd` service instead of `launchd`:

```bash
(sleep 10 && systemctl restart orbit) >/dev/null 2>&1 &
disown
```

## Step 4: Deploy the package

Follow the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) to upload the package from step 1 and the post-install script from step 3 to a fleet, then install it on your hosts.

## Verify the extension is running

- In the Fleet UI, go to **Hosts > (select the host) > Software**, and confirm the package shows as installed.
- Run a live query against `osquery_extensions` (or your extension's custom table) from the **Queries** page, targeted at the host, to confirm the extension loaded and is returning data.

## Step 5: Deploy to your whole fleet

Once you've confirmed the extension works on a test host, use a [policy](https://fleetdm.com/securing/what-are-fleet-policies) to automatically install it everywhere it's missing, instead of installing it host by host.

1. In Fleet, go to the **Policies** tab and add a new policy. Use a query that checks whether the extension binary is already present, so the policy only fails (and triggers an install) on hosts that need it:

   **macOS and Linux:**

   ```sql
   SELECT 1 FROM file WHERE path = '/usr/local/bin/my-custom-extension.ext';
   ```

   **Windows:**

   ```sql
   SELECT 1 FROM file WHERE path = 'C:\Program Files\MyCompany\MyCustomExtension\my-custom-extension.ext.exe';
   ```

   > If you know the name your extension registers with osquery (set in its source code), you can check that it's actually loaded instead of just present on disk: `SELECT 1 FROM osquery_extensions WHERE name = 'your_extension_name';`.

2. Select **Manage automations > Install software**, then select the policy and the package you built in step 1.

Now any host that fails the policy, because the extension isn't installed yet, automatically gets the package installed, which runs the post-install script from step 3 and loads the extension. See the [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet) for retry behavior and how to scope this to specific hosts with labels.

## Troubleshoot

**Extension isn't loading after install.** Check that `extensions.load` (in Orbit's root directory) contains the exact path used in your post-install script, and that the extension file at that path has the ownership and permissions from step 1.

**Orbit doesn't seem to restart.** The restart is scheduled for 10 seconds after the post-install script exits, so it happens after Fleet marks the install successful, not during it. If the extension still isn't loaded a few minutes later, confirm the restart command matches your platform: `launchctl kickstart -k system/com.fleetdm.orbit` on macOS, `systemctl restart orbit` on Linux, or `Restart-Service -Name "Fleet osquery"` on Windows.

## Considerations and best practices

### Security

When selecting custom extensions to deploy, prioritize open source solutions whenever possible. Open source extensions provide transparency, allowing you to audit the code for security vulnerabilities and ensure trustworthiness before deployment.

### Version management

Implement versioning in your detection policies and extension names to handle updates.

## Custom extension examples

Here are some examples of the custom extensions we use at Fleet:

- [macos_compatibility](https://github.com/harrisonravazzolo/macos-compatibility-ext): Get a snapshot of what version of macOS your MacBooks are compatible with.
- [snap_packages](https://github.com/allenhouchins/fleet-stuff/tree/main/linux-mdm-snap-packages): Collect packages installed by snap in a similar syntax to what you are used to with deb_packages or rpm_packages.

<meta name="articleTitle" value="Deploying custom osquery extensions in Fleet: A step-by-step guide">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-06">
<meta name="description" value="Learn how to package, test, and deploy a custom osquery extension in Fleet, with the loader and Orbit restart folded into one install.">
