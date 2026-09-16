# Deploy Visual Studio on Windows with Fleet

Visual Studio doesn't install like most Windows software. Adding the Fleet-maintained app and letting it run gives every host the core IDE shell. That shell has no workloads, so nobody can build anything with it. This guide covers choosing a workload strategy, deploying it, and letting developers pick their own workloads. It applies to Visual Studio 2022 Community, Professional, and Enterprise on Windows.

## Prerequisites

Check these before you start:

- Fleet with Windows hosts enrolled
- Admin or maintainer access to the fleet you're deploying to
- A GitOps repository, if you plan to pin workloads or set installer policy
- License entitlement for Professional or Enterprise. Community is free, and its license covers classroom and academic use.
- Hosts with network access to Microsoft's download servers

> **Warning:** Visual Studio downloads its payload during installation. Install time depends on the host's internet connection and the workloads you select. Fleet stops an install script after one hour, and that download counts against the hour. Test your selection on a host with a typical connection before you roll it out.

## Understand what the default install gives you

The Fleet-maintained app runs the Visual Studio bootstrapper unattended. With no workload selected, that installs the core shell only. The core install downloads far less than one that includes workloads, so it finishes sooner and uses less disk space.

That core install is still useful, because it includes the Visual Studio Installer. That's the app developers use to add workloads later.

Decide which you need before you deploy:

- Everyone on a fleet gets the same workloads. Pin them in the install script.
- Developers choose their own workloads. Install the core, then let them use the Visual Studio Installer.

You can mix the two. Pin a baseline workload set, and still let developers add to it.

## Add the Fleet-maintained app

1. Go to **Software** and choose your fleet.
2. Select **Add software**, then open the **Fleet-maintained** tab.
3. Select the edition you want: **Visual Studio Community 2022**, **Visual Studio Professional 2022**, or **Visual Studio Enterprise 2022**.
4. Choose whether to install it automatically or offer it as self-service, then add the app.

In GitOps, add the slug for the edition instead:

```yaml
software:
  fleet_maintained_apps:
    - slug: visual-studio-2022-professional/windows
```

> **Note:** Each edition is a separate app. A host can run more than one edition at once. Add only the editions you plan to deploy.

## Pin workloads for a fleet

Use this when everyone on a fleet needs the same setup. Override the install script with your workload selection.

Save a script like this in your GitOps repository:

```powershell
$exeFilePath = "${env:INSTALLER_PATH}"

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--quiet --wait --norestart --add Microsoft.VisualStudio.Workload.ManagedDesktop --includeRecommended"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# 3010 and 1641 mean the install succeeded and a reboot is pending.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install succeeded, reboot required to finish"
  Exit 0
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode
```

Point the app at it:

```yaml
software:
  fleet_maintained_apps:
    - slug: visual-studio-2022-professional/windows
      install_script:
        path: ../lib/software/vs-professional-install.ps1
```

Repeat `--add` for each workload you want. See Microsoft's [workload and component IDs](https://learn.microsoft.com/en-us/visualstudio/install/workload-and-component-ids) for the full list.

> **Note:** `--wait` belongs on the bootstrapper, which is what this script runs. Without it, the bootstrapper starts the real install in the background and returns before the install finishes.

For a larger selection, export a configuration from a machine you've already set up. Open the Visual Studio Installer, select **More**, then **Export configuration**. Write that file's contents to disk in your install script and pass `--config` instead of repeating `--add`.

## Let developers pick their own workloads

Standard users can't run the Visual Studio Installer silently. Microsoft blocks `--quiet` and `--passive` for them no matter how you configure the machine. They can use the Visual Studio Installer interface, though, once you allow it.

Set the policy with a post-install script:

```powershell
# 2 gives standard users all Visual Studio Installer functionality,
# including Modify. 1 allows updates and rollback only.
$key = 'HKLM:\SOFTWARE\Policies\Microsoft\VisualStudio\Setup'

try {
  if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
  New-ItemProperty -Path $key -Name 'AllowStandardUserControl' -Value 2 -PropertyType DWord -Force | Out-Null
  New-ItemProperty -Path $key -Name 'HideAvailableTab' -Value 1 -PropertyType DWord -Force | Out-Null
  Write-Host 'AllowStandardUserControl set to 2'
  Exit 0
} catch {
  Write-Host "Error: $_"
  Exit 1
}
```

```yaml
software:
  fleet_maintained_apps:
    - slug: visual-studio-2022-professional/windows
      post_install_script:
        path: ../lib/software/vs-allow-standard-user-control.ps1
```

> **Warning:** `AllowStandardUserControl` set to `2` also lets standard users install other Visual Studio products from the **Available** tab. The script above sets `HideAvailableTab` to hide that tab. Remove that line if you want users to install other products themselves.

Tell developers how to use it:

1. Open **Visual Studio Installer** from the Start menu.
2. Select **Modify** on the installed edition.
3. Choose the workloads to add.
4. Select **Modify** again to apply.

> **Note:** After this point, Fleet can't change workloads for standard users on its own. Silent installer commands stay blocked for them. Adding and removing workloads happens in the Visual Studio Installer.

## Verify

1. Go to the host's **Host details** page, open the **Software** tab, and confirm the edition appears with a version.
2. On the host, open **Visual Studio Installer** and confirm the workloads you expect are installed.
3. If you set the installer policy, sign in as a standard user and confirm **Modify** works without an administrator prompt.

## Troubleshoot

**The install fails after about an hour**

Fleet stops install scripts at one hour, and the payload downloads inside that window. Reduce the workloads you pin, or drop `--includeRecommended`. Check the host's network speed against the download size.

**Developers see an administrator prompt in the Visual Studio Installer**

The policy either didn't apply or is set to `1`. Confirm `AllowStandardUserControl` is `2` under `HKLM\SOFTWARE\Policies\Microsoft\VisualStudio\Setup`. The policy also needs a current Visual Studio Installer on the host, which the Fleet-maintained app installs.

**Uninstall fails while Visual Studio is open**

Visual Studio won't uninstall while the IDE is running. Close Visual Studio on the host and retry.

**A host with two editions shows software as missing**

Windows renames the entry when a host has more than one Visual Studio instance. The second one appears as `Visual Studio Professional 2022 (2)`. The Fleet-maintained apps account for this. If you deploy Visual Studio as a custom package instead, your detection query needs to match the renamed entry too.

## Further reading

- [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps)
- Microsoft's [command-line parameters for Visual Studio installs](https://learn.microsoft.com/en-us/visualstudio/install/use-command-line-parameters-to-install-visual-studio)
- Microsoft's [enterprise deployment policies](https://learn.microsoft.com/en-us/visualstudio/install/configure-policies-for-enterprise-deployments)

<meta name="articleTitle" value="Deploy Visual Studio on Windows with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-08-06">
<meta name="category" value="guides">
<meta name="description" value="Deploy Visual Studio 2022 on Windows with Fleet. Pin workloads for a fleet, or let developers choose their own.">
