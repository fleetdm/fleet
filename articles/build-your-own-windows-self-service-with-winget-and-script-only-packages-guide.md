# Add Microsoft Store apps to Windows with script-only packages

You can install Microsoft Store apps for all users on Windows hosts with a script-only package. No user needs to be signed in. Only free apps are supported. Paid apps fail to install with a licensing error. Built-in support for Microsoft Store apps is [coming soon](https://github.com/fleetdm/fleet/issues/43493).

## Step 1: Find the Store ID and package family name

You need two identifiers.

1. Find the app on the [Microsoft Store website](https://apps.microsoft.com). The **Store ID** is the last part of the URL. For Company Portal (`https://apps.microsoft.com/detail/9WZDNCRFJ3PZ`), it's `9WZDNCRFJ3PZ`.

2. Get the **package family name**. Run one of these, replacing the Store ID.

   On Windows (PowerShell):

   ```powershell
   (Invoke-RestMethod "https://storeedgefd.dsx.mp.microsoft.com/v9.0/packageManifests/9WZDNCRFJ3PZ?Market=US").Data.Versions.Installers |
       Select-Object InstallerType, PackageFamilyName -Unique
   ```

   On macOS or Linux (requires [jq](https://jqlang.org)):

   ```sh
   curl -s "https://storeedgefd.dsx.mp.microsoft.com/v9.0/packageManifests/9WZDNCRFJ3PZ?Market=US" |
     jq '[.Data.Versions[].Installers[] | {InstallerType, PackageFamilyName}] | unique'
   ```

   For Company Portal, `InstallerType` is `msstore` and `PackageFamilyName` is `Microsoft.CompanyPortal_8wekyb3d8bbwe`.

> **Note:** The scripts work for apps whose `InstallerType` is `msstore`. For other apps, use a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) or [custom package](https://fleetdm.com/guides/deploy-software-packages).

## Step 2: Get the install script

1. Download [`install-microsoft-store-app.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/install-microsoft-store-app.ps1).
2. Set `$storeId` and `$packageFamilyName` at the top of the script to the identifiers from Step 1.
3. Save it with a name for the app, such as `install-company-portal.ps1`.

To update the app later, select **Reinstall** in Fleet.

## Step 3: Get the uninstall script

1. Download [`uninstall-microsoft-store-app.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/uninstall-microsoft-store-app.ps1).
2. Set `$packageFamilyName` at the top of the script to the package family name from Step 1.
3. Save it with a name for the app, such as `uninstall-company-portal.ps1`.

## Step 4: Add the package to Fleet

Add the install script as a script-only package, with the uninstall script from Step 3.

### Fleet UI

1. Go to **Software** and select the fleet.
2. Select **Add software > Custom package** and upload `install-company-portal.ps1`.
3. Expand **Advanced options** and paste the uninstall script into **Uninstall script**.
4. Optionally, turn on **Self-service**.
5. Select **Add software**.

### GitOps

Save both scripts under `lib/windows/scripts/` in your GitOps repo. Then add the package to `fleets/<name>.yml`:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
```

## Verify

1. Install the app on a test host from **Host details > Software**.
2. Open the install's details. The output ends with "Installed version … for all users."
3. Refetch the host. The app appears in the host's software inventory.
4. Sign in as another user on the host. The app is in their Start menu. On a user's first sign-in after the install, it can take a few seconds to appear.

## Troubleshoot

**The install fails with "The Microsoft Store couldn't license this app."** The app is paid. Only free apps are supported.

**The install fails with "The app is open and couldn't be closed."** Ask the user to close the app, then select **Reinstall**.

**The install fails with "Timed out waiting for the Microsoft Store."** The host couldn't finish the download within 4 minutes. Check the host's network connection, then try again.

**The install fails with "The app isn't installed for all users even though the install reported success."** Check that `$packageFamilyName` in the install script matches the package family name from Step 1.

## Further reading

- [Deploy software packages](https://fleetdm.com/guides/deploy-software-packages)
- [Scripts](https://fleetdm.com/guides/scripts)

<meta name="articleTitle" value="Add Microsoft Store apps to Windows with script-only packages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Install free Microsoft Store apps for all users on Windows hosts with Fleet script-only packages, no winget or signed-in user needed.">
