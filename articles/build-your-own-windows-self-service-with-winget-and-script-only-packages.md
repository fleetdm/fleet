# Build your own Windows self-service with winget

_The same Fleet 4.89.0 additions that turned `apt-get install` into a Linux self-service catalog work on Windows with `.ps1` packages. Windows has one problem Linux doesn't: Microsoft says winget isn't supported in the SYSTEM context._

## Key takeaways

- **A `.ps1` script-only package is a full install lifecycle.** As of Fleet 4.89.0, a PowerShell script-only package can carry an uninstall script, a pre-install query, and a post-install script, which is everything you need to put a package manager behind a self-service tile.
- **Fleet runs your script as SYSTEM, and winget does not officially run there.** App Installer ships as an MSIX package that can be registered for any user except `NT AUTHORITY\SYSTEM`, so `winget` is not on the PATH your script inherits. You resolve the executable yourself.
- **Machine-scope apps from the `winget` source work well; Microsoft Store apps do not.** The community `winget` source installs machine-wide cleanly. The `msstore` source refuses device-wide installs outright, so a "Store app" tile needs a different approach.
- **For per-user apps, hand the install to the logged-on user.** A short-lived scheduled task running as the console user is the same pattern Fleet uses for its own per-user Windows apps, and it's what makes user-scope winget installs possible at all.
- **PowerShell won't fail the install for you.** Unlike `set -euo pipefail` in bash, PowerShell ignores a native command's non-zero exit code, so you have to propagate it yourself. And winget's "already installed" code isn't a failure.
- **The per-app work is mechanical enough to generate.** Given a winget package ID, a generator emits the install script, the uninstall script, and the YAML block, so adding an app is one command and a pull request.

<a purpose="cta-button" href="https://fleetdm.com/infrastructure-as-code">See it managed as code</a>

[Fleet 4.89.0](https://fleetdm.com/releases/fleet-4.89.0) added an uninstall script, a pre-install query, and a post-install script to script-only packages. On Linux, that was enough to [build a self-service catalog on top of apt and dnf](https://fleetdm.com/articles/build-your-own-linux-self-service-with-script-only-packages) with no `.deb` files to host. Windows gets the same three additions for `.ps1` files, and Windows has a package manager of its own in winget.

The recipe does not port over unchanged, though. On Linux, Fleet hands your script to a root shell and `apt-get` is right there on the PATH. On Windows, Fleet hands your script to PowerShell running as SYSTEM, and that is precisely the one account winget cannot be registered for. Everything interesting about building this on Windows follows from that one fact.

## The building block: script-only packages on Windows

A script-only package is a `.sh` file (Linux) or a `.ps1` file (Windows) that you add to Fleet as software. There is no installer binary and no metadata extraction. The file's contents are the install script, and Fleet runs them on the host when someone installs the "package."

Since 4.89.0, a script-only package can also carry a pre-install query that must return at least one row before the install runs, a post-install script whose non-zero exit fails the install and triggers the uninstall, and an uninstall script that runs when an admin or end user removes the software. What it still cannot do is take an `install_script` key, because the file's contents already are the install script, or install automatically through a policy. Drive these through self-service.

That's the shape of a package manager front end: put an app on, take it off, check your work.

## Where the Linux recipe breaks: winget and the SYSTEM context

Fleet's agent executes Windows scripts as `powershell -MTA -ExecutionPolicy Bypass -File <script>`, running as SYSTEM. Microsoft's own [winget troubleshooting documentation](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) is unambiguous about what that means: because winget is delivered through App Installer as a packaged MSIX application, and MSIX packages can be registered for any user except `NT AUTHORITY\SYSTEM`, the winget CLI is not supported in the system context.

In practice that produces two separate problems, and they need separate answers.

The first is cosmetic but fatal: the `winget` app execution alias lives in a per-user path, so SYSTEM has no `winget` command. The binary is still on disk inside the App Installer package directory, and calling it there works for machine-scope installs. This is the workaround the Windows admin community has converged on, and it's the one used below. Microsoft also notes that the `Microsoft.WinGet.Client` PowerShell module can be used in the system context for applications installed machine-wide, which is the supported path if you're willing to get that module onto every host first.

The second problem is structural, and it's the one that matters if you came here wanting Microsoft Store apps. Store packages are inherently per-user, and winget's `msstore` source rejects device-wide installs with "Device wide install for msstore type is not supported under admin context." No amount of path resolution fixes that, because it isn't a path problem. If the app you want exists in the community `winget` source, use that source and read the next section. If it only exists in the Store, skip ahead to running the install as the logged-on user, and expect Store licensing and Entra ID authentication to complicate it further.

## The install script: machine-scope apps

Start by finding the executable. The App Installer directory name embeds the version, so sort on the parsed version rather than the string. A plain descending sort puts `1.9` above `1.24`.

```powershell
# install-7zip.ps1
$PackageId = "7zip.7zip"

$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path

if (-not $winget) {
    Write-Host "App Installer (winget) was not found on this host."
    exit 1
}

Write-Host "Installing $PackageId with $winget"
& $winget install --id $PackageId --exact --source winget `
    --scope machine --silent `
    --accept-package-agreements --accept-source-agreements `
    --disable-interactivity
$code = $LASTEXITCODE

# 0 = installed. -1978335135 (0x8A150061) = already installed, which is not a failure.
if ($code -eq 0 -or $code -eq -1978335135) {
    Write-Host "$PackageId is installed."
    exit 0
}

Write-Host "winget exited with $code"
exit 1
```

Three details in there are doing real work.

`--exact` with `--id` stops winget from resolving a fuzzy name match to the wrong package, which is a genuine risk when nobody is watching the output. `--silent`, `--accept-package-agreements`, `--accept-source-agreements`, and `--disable-interactivity` together are the Windows equivalent of `DEBIAN_FRONTEND=noninteractive`: without them a prompt waits forever for a user who isn't there.

The explicit exit code handling is the part with no Linux counterpart. In bash, `set -euo pipefail` makes the script die the moment a command fails. PowerShell does not work that way. `$ErrorActionPreference` governs PowerShell errors, not native process exit codes, so a failed `winget install` leaves your script running happily and exiting 0. Fleet reads that 0 and marks the install successful. You have to check `$LASTEXITCODE` and propagate it, and while you're there, treat `-1978335135` as success, because a host that already has the app is not a failed install.

One honest caveat on `--scope machine`: Microsoft [documents](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) that scope is reliable for MSI and MSIX packages but not deterministic for EXE-based installers, where the arguments to specify scope may not exist at all. Verify per app rather than assuming.

## Add the uninstall

The mirror image, with the same path resolution and the same care about exit codes. Removing something that was already absent is a successful no-op, so `-1978335212` (no packages found) exits 0.

```powershell
# uninstall-7zip.ps1
$PackageId = "7zip.7zip"

$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path

if (-not $winget) {
    Write-Host "App Installer (winget) was not found on this host."
    exit 1
}

& $winget uninstall --id $PackageId --exact --silent `
    --accept-source-agreements --disable-interactivity
$code = $LASTEXITCODE

# -1978335212 (0x8A150014) = no packages found, so there is nothing to remove.
if ($code -eq 0 -or $code -eq -1978335212) {
    Write-Host "$PackageId is not installed."
    exit 0
}

Write-Host "winget exited with $code"
exit 1
```

Save both under `lib/windows/scripts/` in your GitOps repo, then register the package in the fleet where you want it available. Software is defined per fleet, so this goes in `fleets/<name>.yml` or `fleets/unassigned.yml`:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-7zip.ps1
      display_name: 7-Zip
      self_service: true
      categories:
        - "💻 Productivity"
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-7zip.ps1
```

With `self_service: true`, the app appears on the end user's self-service page, reachable from the Fleet icon in the Windows system tray. When they click install, Fleet's agent runs the install script as SYSTEM. When they remove it, the uninstall script runs.

## Per-user apps: hand the install to the logged-on user

Some apps refuse to install machine-wide at all. Squirrel-based Electron apps, anything from the Store, and a long tail of `--scope user` packages all fall in this bucket, and no SYSTEM-context trick makes them work, because the install genuinely needs a user profile to land in.

The answer is to stop fighting it and run winget as the user. Fleet uses exactly this pattern for its own per-user Windows [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps): create a short-lived scheduled task that runs as the owner of the `explorer.exe` process, start it, wait for it to finish, then unregister it.

```powershell
# install-figma-user-scope.ps1
$PackageId = "Figma.Figma"
$exitCode = 0
$taskName = "fleet-install-$PackageId"

try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner).User
    if (-not $userName) { throw "No logged-on user found; cannot install a user-scope app." }

    $args = "install --id $PackageId --exact --source winget --scope user --silent " +
            "--accept-package-agreements --accept-source-agreements --disable-interactivity"
    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $args
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
    $task = New-ScheduledTask -Action $action -Settings $settings

    Register-ScheduledTask $taskName -InputObject $task -User $userName | Out-Null
    Start-ScheduledTask -TaskName $taskName -TaskPath "\"

    $startDate = Get-Date
    do {
        Start-Sleep -Seconds 5
        $state = (Get-ScheduledTask -TaskName $taskName).State
        Write-Host "Scheduled task is '$state'."
        if ((New-TimeSpan -Start $startDate -End (Get-Date)).TotalSeconds -gt 600) {
            throw "Timed out waiting for the install to finish."
        }
    } while ($state -eq "Running" -or $state -eq "Queued")
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
}

exit $exitCode
```

Because the task runs in the user's session, plain `winget.exe` resolves normally through the app execution alias and no path hunting is needed. The trade-off is real, though: this only works when someone is logged in, the scheduled task's own exit code doesn't come back to you, and you learn nothing about whether winget actually succeeded. That makes the verification step in the next section mandatory here rather than merely advisable.

The same pattern also cleanly covers the uninstall, swapping `install` for `uninstall`. Sweep every user profile if the app may have been installed by more than one person.

## Guardrails: targeting and verification

### Target the right hosts with labels

Define a dynamic label and reference it on the package so only the intended hosts see the tile:

```yaml
labels:
  - name: Windows
    query: "SELECT 1 FROM os_version WHERE platform = 'windows';"
    label_membership_type: dynamic
```

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-7zip.ps1
      display_name: 7-Zip
      self_service: true
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-7zip.ps1
```

Any label you reference on a package has to be defined in the `labels` section first, and you can use only one of `labels_include_any`, `labels_include_all`, or `labels_exclude_any` per package. Labels also give you somewhere to put the hosts winget can't help: Windows Server and LTSC images often ship without App Installer at all, and excluding them is kinder than letting the script fail on every attempt.

### Verify the install actually landed

A post-install script runs after the install, and a non-zero exit fails the install and triggers your uninstall script. Check the state of the machine rather than asking winget again, so a broken winget can't vouch for itself:

```powershell
# verify-7zip.ps1
$DisplayNamePattern = "7-Zip*"

$paths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)

$found = Get-ItemProperty -Path $paths -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -like $DisplayNamePattern }

if (-not $found) {
    Write-Host "$DisplayNamePattern was not found in the uninstall registry."
    exit 1
}

Write-Host "Found $($found[0].DisplayName) $($found[0].DisplayVersion)."
exit 0
```

```yaml
      post_install_script:
        path: ../lib/windows/scripts/verify-7zip.ps1
```

For a user-scope install, `HKLM` is the wrong place to look. Those entries land under the user's own hive, so enumerate `HKEY_USERS` from SYSTEM instead of `HKCU`, which as SYSTEM points at the wrong profile entirely.

A pre-install query can gate the install as well, proceeding only if the query returns a row. Labels are the better tool for platform targeting, so save the query for a genuine precondition.

## Wire it into GitOps

Every self-service app is a set of files in your repository:

```
lib/
  windows/
    scripts/
      install-7zip.ps1
      uninstall-7zip.ps1
      verify-7zip.ps1
fleets/
  workstations.yml   # references the scripts under software.packages
```

Adding, changing, or removing an app is a pull request. Reviewers see the exact commands that will run as SYSTEM on every targeted host, the change ships through CI when it merges, and the catalog is auditable and reversible by design. [Turn on GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) to make the Fleet UI reflect that these are code-managed. Fleet manages its own software this way, and the [it-and-security configuration](https://github.com/fleetdm/fleet/tree/main/it-and-security/fleets) is public if you want a real-world layout to borrow from.

## Automate it: from a winget ID to a self-service tile

Once the pattern settles, the only real input is the package ID. This generator writes both scripts and prints the YAML block:

```powershell
# New-SelfServiceApp.ps1 -PackageId 7zip.7zip -DisplayName "7-Zip"
param(
    [Parameter(Mandatory)][string]$PackageId,
    [string]$DisplayName = $PackageId,
    [string]$Category = "💻 Productivity"
)

$dir = "lib/windows/scripts"
New-Item -ItemType Directory -Path $dir -Force | Out-Null
$slug = $PackageId.ToLower().Replace(".", "-")

$resolve = @'
$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path
if (-not $winget) { Write-Host "App Installer (winget) was not found."; exit 1 }
'@

@"
`$PackageId = "$PackageId"
$resolve
& `$winget install --id `$PackageId --exact --source winget --scope machine --silent ``
    --accept-package-agreements --accept-source-agreements --disable-interactivity
`$code = `$LASTEXITCODE
if (`$code -eq 0 -or `$code -eq -1978335135) { exit 0 }
Write-Host "winget exited with `$code"
exit 1
"@ | Set-Content "$dir/install-$slug.ps1" -Encoding UTF8

@"
`$PackageId = "$PackageId"
$resolve
& `$winget uninstall --id `$PackageId --exact --silent ``
    --accept-source-agreements --disable-interactivity
`$code = `$LASTEXITCODE
if (`$code -eq 0 -or `$code -eq -1978335212) { exit 0 }
Write-Host "winget exited with `$code"
exit 1
"@ | Set-Content "$dir/uninstall-$slug.ps1" -Encoding UTF8

@"

# Add to your fleet's software.packages:
    - path: ../$dir/install-$slug.ps1
      display_name: $DisplayName
      self_service: true
      categories:
        - "$Category"
      uninstall_script:
        path: ../$dir/uninstall-$slug.ps1
"@
```

Onboarding an app becomes: find the ID with `winget search`, run `./New-SelfServiceApp.ps1 -PackageId 7zip.7zip -DisplayName "7-Zip"`, paste the printed block into your fleet file, and open a pull request. Write the verification script by hand, because the display name in the registry rarely matches the winget ID closely enough to generate.

## The point

Fleet didn't ship a Windows software store in 4.89.0. It shipped three small additions to script-only packages, and those are enough to build one on top of the package manager Microsoft already ships, with no installers to host and no per-app UI work.

Windows makes you earn it in a way Linux doesn't. The SYSTEM context is a real constraint, not a bug you can wait out, and it forces a decision per app: machine scope through the resolved binary, or user scope through the logged-on user. Make that decision once, encode it in a script, and the rest is a naming convention in Git.

## See it live

- Follow the [step-by-step guide](https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide) to build your first app end to end.
- Read the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for the full detail on script-only packages, pre-install queries, and uninstall scripts.
- Get a demo: [fleetdm.com/contact](https://fleetdm.com/contact).
- Join a free GitOps workshop: [fleetdm.com/workshops](https://fleetdm.com/workshops).

_Managing devices as code, one pull request at a time. Start with the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files) or [talk to us](https://fleetdm.com/contact)._

<meta name="articleTitle" value="Build your own Windows self-service with winget">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-08-05">
<meta name="description" value="Use Fleet script-only .ps1 packages to put winget behind a Windows self-service tile, including the SYSTEM context workaround.">
