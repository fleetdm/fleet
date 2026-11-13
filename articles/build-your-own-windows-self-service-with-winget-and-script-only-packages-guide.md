# Add Microsoft Store apps to Windows with script-only packages

You can install Microsoft Store apps on Windows hosts, for all users, with a script-only package. The install script runs as SYSTEM and calls the Microsoft Store install service, the same one the `winget` CLI uses, so no user needs to be signed in. Built-in support for Microsoft Store apps is [coming soon](https://github.com/fleetdm/fleet/issues/43493).

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script, pre-install query, and post-install script on script-only packages were added in this release.
- Windows hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed Fleet's agent without `--enable-scripts`.
- A free Microsoft Store app. Paid apps fail to install with a licensing error.

> **Note:** The Microsoft Store install service is documented for Microsoft's own apps, such as winget, and isn't officially supported for third parties. Microsoft could change it.

## Step 1: Find the Store ID and package family name

You need two identifiers.

1. Find the app on the [Microsoft Store website](https://apps.microsoft.com). The **Store ID** is the last part of the URL. For Company Portal (`https://apps.microsoft.com/detail/9WZDNCRFJ3PZ`), it's `9WZDNCRFJ3PZ`.

2. Get the **package family name** and confirm the app is supported. Run this on any computer, replacing the Store ID:

   ```powershell
   (Invoke-RestMethod "https://storeedgefd.dsx.mp.microsoft.com/v9.0/packageManifests/9WZDNCRFJ3PZ?Market=US").Data.Versions.Installers |
       Select-Object InstallerType, PackageFamilyName -Unique
   ```

   For Company Portal, `InstallerType` is `msstore` and `PackageFamilyName` is `Microsoft.CompanyPortal_8wekyb3d8bbwe`.

> **Note:** The scripts in this guide are written for MSIX apps: `InstallerType` is `msstore`, and the Store ID starts with `9`. Apps that use their own installer (MSI or EXE) have Store IDs that start with `XP`. They have no package family name, and their installer decides whether to install for one user or all users, so the scripts' all-users check fails for them. You can adapt the scripts, or add these apps as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) or [custom package](https://fleetdm.com/guides/deploy-software-packages).

## Step 2: Get the install script

1. Download [`install-microsoft-store-app.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/install-microsoft-store-app.ps1).
2. Set `$storeId` and `$packageFamilyName` at the top of the script to the identifiers from Step 1.
3. Save it with a name for the app, such as `install-company-portal.ps1`.

The script installs the app for all users on the host, including users who sign in later. If the app is already installed, it installs the latest version, so selecting **Reinstall** in Fleet updates the app. If the app is open during an update, the Microsoft Store closes it, and it doesn't reopen on its own.

## Step 3: Get the uninstall script

1. Download [`uninstall-microsoft-store-app.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/uninstall-microsoft-store-app.ps1).
2. Set `$packageFamilyName` at the top of the script to the package family name from Step 1.
3. Save it with a name for the app, such as `uninstall-company-portal.ps1`.

The script removes the app for all users. If the app is open, its window can stay open until the user closes it.

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

> **Note:** Script-only packages don't support automatic install through a policy. Install them from the host's **Host details** page or through self-service.

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

**`InstallerType` in Step 1 isn't `msstore`.** The app uses its own installer, and the scripts' all-users check fails for it. Adapt the scripts, or add the app as a Fleet-maintained app or custom package.

## Further reading

- [Deploy software packages](https://fleetdm.com/guides/deploy-software-packages)
- [Scripts](https://fleetdm.com/guides/scripts)

<meta name="articleTitle" value="Add Microsoft Store apps to Windows with script-only packages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Install free Microsoft Store apps for all users on Windows hosts with Fleet script-only packages, no winget or signed-in user needed.">
