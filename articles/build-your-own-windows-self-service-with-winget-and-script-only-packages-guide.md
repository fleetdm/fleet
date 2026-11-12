# Add Microsoft Store apps to Windows with a script-only package

You can install Microsoft Store apps on Windows hosts for every user with a script-only package. The script runs as SYSTEM and uses a Microsoft service to install the app, so it doesn't need a logged-in user. Built-in support for Microsoft Store apps is [coming soon](https://github.com/fleetdm/fleet/issues/43493).

> **TODO (@marko-lisica):** Name the Microsoft service the script uses and link to Microsoft's docs for it.

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script on script-only packages was added in this release.
- Windows hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed Fleet's agent without `--enable-scripts`.

## Step 1: Find the Store ID

Every Store app has a Store ID, a twelve-character string like `9WZDNCRFJ3PZ` for Company Portal. Find it with winget on any Windows computer:

```powershell
winget search --source msstore "company portal"
```

You can also copy it from the end of the app's URL on [apps.microsoft.com](https://apps.microsoft.com).

## Step 2: Write the install script

Save this as `install-company-portal.ps1`, and set `$StoreId` to the ID from Step 1.

```powershell
$StoreId = "9WZDNCRFJ3PZ"

# TODO (@marko-lisica): Add the all-users install script.
```

## Step 3: Write the uninstall script

Save this as `uninstall-company-portal.ps1`, and set `$StoreId` to the same ID.

```powershell
$StoreId = "9WZDNCRFJ3PZ"

# TODO (@marko-lisica): Add the all-users uninstall script.
```

## Step 4: Add the package to Fleet

Add the package in the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Software > Add software > Custom package** and upload `install-company-portal.ps1`.
2. Open **Advanced options** and paste the contents of `uninstall-company-portal.ps1` as the uninstall script.
3. To let end users install the app themselves, check **Self-service**.
4. Select **Add software**.

### GitOps

Save both scripts under `lib/windows/scripts/` in your GitOps repo. Then add the package to `fleets/<name>.yml` or `fleets/unassigned.yml`:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
```

With `self_service: true`, the app appears on the end user's self-service page, reachable from the Fleet icon in the Windows system tray.

## Verify

Store apps don't appear in the `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall` registry keys. To confirm an install, run this script on the host from Fleet. Replace `Microsoft.CompanyPortal` with the app's MSIX package name.

```powershell
Get-AppxPackage -AllUsers -Name "Microsoft.CompanyPortal" | Select-Object Name, Version, PackageUserInformation
```

## Further reading

- [Deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for full detail on script-only packages and uninstall scripts.
- [Put Microsoft Store apps on Windows with a script-only package](https://fleetdm.com/articles/build-your-own-windows-self-service-with-winget-and-script-only-packages) for the reasoning behind this approach.
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files).

<meta name="articleTitle" value="Add Microsoft Store apps to Windows with a script-only package">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Use a Fleet script-only .ps1 package to install Microsoft Store apps on Windows hosts for every user.">
