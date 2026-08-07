# Put Microsoft Store apps in Windows self-service with winget

_Installing a Store app with winget is one command. Getting that one command to run from Fleet is the interesting part, because Fleet runs scripts as SYSTEM and the Microsoft Store does not._

## Key takeaways

- **The per-app work is genuinely one line.** A Store app is identified by its Store ID, and the install is `winget install --id <StoreId> --source msstore`. Everything else in the script is boilerplate you write once.
- **That one line fails as SYSTEM, and no flag fixes it.** Fleet hands Windows scripts to PowerShell running as SYSTEM. The `msstore` source refuses device-wide installs outright, because Store packages are per-user by design.
- **The fix is to run winget in the user's session, not to run it harder.** A short-lived scheduled task owned by the console user puts winget where the Store expects it, and `winget` resolves normally there with no path hunting.
- **Verification has to read the MSIX world, not the registry.** Store apps never appear in the uninstall registry keys, so `Get-AppxPackage -AllUsers` is the check that tells you the truth.
- **Machine-wide Store deployment exists, but it's a different tool.** Downloading the package and provisioning it with `Add-AppxProvisionedPackage` works as SYSTEM, at the cost of Entra ID licensing rights and a file to host.
- **A Store ID is enough to generate the whole tile.** Given an ID, a generator emits the install script, the uninstall script, and the YAML, so onboarding an app is one command and a pull request.

<a purpose="cta-button" href="https://fleetdm.com/infrastructure-as-code">See it managed as code</a>

[Fleet 4.89.0](https://fleetdm.com/releases/fleet-4.89.0) added an uninstall script, a pre-install query, and a post-install script to script-only packages. On Linux, that was enough to [build a self-service catalog on top of apt and dnf](https://fleetdm.com/articles/build-your-own-linux-self-service-with-script-only-packages). Windows gets the same three additions for `.ps1` files, and Windows already ships a package manager that can reach the Microsoft Store.

So the obvious move is a two-line script: `winget install` on the way in, `winget uninstall` on the way out. Those are the right commands. They won't run where Fleet puts them, though, and understanding why is what turns a broken tile into a working one.

## What the install and uninstall actually are

Start with the destination, because it's short. Every Store app has a Store ID, a twelve-character string like `9WZDNCRFJ3PZ` for Company Portal. Find it with a search:

```powershell
winget search --source msstore "company portal"
```

Given the ID, the two commands you need are these:

```powershell
winget install --id 9WZDNCRFJ3PZ --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity
winget uninstall --id 9WZDNCRFJ3PZ --accept-source-agreements --disable-interactivity
```

Pin to the ID rather than the name. A name match can resolve to the wrong package, and nobody is watching the output when Fleet runs this. The flags matter as much as the ID: winget [prompts for msstore source agreements even on uninstall](https://github.com/microsoft/winget-cli/issues/1736), and in an unattended script that prompt is a permanent hang. `--disable-interactivity` turns any prompt these flags miss into a failure instead. Everything from here on exists to get those two commands executed in a place where they work.

## Why the one-liner fails as SYSTEM

Fleet's agent executes Windows scripts as `powershell -MTA -ExecutionPolicy Bypass -File <script>`, running as SYSTEM. That collides with the Store in two separate ways.

The first is that winget itself isn't there. Microsoft's [winget troubleshooting documentation](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) states it plainly: winget ships through App Installer as a packaged MSIX application, MSIX packages can be registered for any user except `NT AUTHORITY\SYSTEM`, and so the winget CLI is not supported in the system context. The `winget` command simply doesn't resolve.

The second is the one that matters here, and it survives every workaround. Even with the binary located, the `msstore` source rejects a device-wide install with "Device wide install for msstore type is not supported under admin context." Store packages are per-user by design, and there is [no supported way](https://github.com/microsoft/winget-cli/issues/3553) to install a user-scoped package as SYSTEM on behalf of a specific user. Asking for `--scope machine` doesn't help either: [it installs the package for the SYSTEM account](https://github.com/microsoft/winget-cli/issues/4748) instead of provisioning it, which is worse than failing, because it looks like it worked.

Running winget in the system context in the first place remains [an open feature request](https://github.com/microsoft/winget-pkgs/issues/346975), not a solved problem you can flag your way around. So stop trying to install as SYSTEM.

## Run winget in the user's session

The way through is to let SYSTEM do what SYSTEM is good at, which is creating a scheduled task, and let the logged-on user do the install. Register a task owned by the owner of the `explorer.exe` process, start it, wait for it to finish, then remove it. Fleet uses this same pattern for its own per-user Windows [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps).

```powershell
# install-company-portal.ps1
$StoreId = "9WZDNCRFJ3PZ"
$WingetArgs = "install --id $StoreId --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity"

$exitCode = 0
$taskName = "fleet-store-$StoreId"

try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner | Select-Object -First 1).User
    if (-not $userName) { throw "No logged-on user, so there is no session to install into." }

    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $WingetArgs
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
            throw "Timed out waiting for winget to finish."
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

Only the first two lines change per app. Everything below them is the same in every script, which is what makes this generatable later.

Because the task runs inside the user's session, plain `winget.exe` resolves through the app execution alias and the Store gets the user context it wants. The uninstall is the identical file with one line different:

```powershell
$WingetArgs = "uninstall --id $StoreId --accept-source-agreements --disable-interactivity"
```

There are two limitations to be aware of with this approach. It needs somebody logged in, which is reasonable for self-service, since a user is clicking the tile. And the scheduled task's exit code doesn't come back to your script, so the script reports success as long as the task ran, whether or not winget did anything. That second one is why the next section isn't optional.

## Verify against the MSIX world, not the registry

If you have built self-service tiles for ordinary Windows installers before, the instinct is to check `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`. That is the wrong place for a Store app. MSIX packages don't register there at all, so a registry check fails on a perfectly good install.

Ask the MSIX subsystem instead. Fleet runs the post-install script as SYSTEM, and SYSTEM is elevated, so `-AllUsers` is available and will see a package installed in a user's profile:

```powershell
# verify-company-portal.ps1
$PackageName = "Microsoft.CompanyPortal"

$pkg = Get-AppxPackage -AllUsers -Name $PackageName -ErrorAction SilentlyContinue
if (-not $pkg) {
    Write-Host "$PackageName is not installed for any user."
    exit 1
}

Write-Host "Found $($pkg[0].Name) $($pkg[0].Version)."
exit 0
```

The `Name` here is the MSIX package name, which is not the Store ID. Get it after a manual install with `Get-AppxPackage | Select-Object Name, PackageFamilyName`, or by matching on a wildcard the first time through.

A non-zero exit from the post-install script fails the install and triggers your uninstall script, so this check is what converts "the scheduled task ran" into "the app is actually there."

## If you need it machine-wide

Sometimes per-user isn't acceptable and the app has to be on the image for everybody. Store apps can be deployed that way, but not through `winget install`. The route is to download the package once and provision it.

On an admin workstation, download the package and its license:

```powershell
winget download --id 9WZDNCRFJ3PZ --source msstore --download-directory C:\staging
```

Then provision it on the host, which does work as SYSTEM:

```powershell
Add-AppxProvisionedPackage -Online -PackagePath .\app.msixbundle -LicensePath .\9WZDNCRFJ3PZ_License.xml
```

There is one important detail to keep in mind with this approach. Downloading a Store package's license file [requires Entra ID authentication](https://learn.microsoft.com/en-us/windows/package-manager/winget/download) by an account holding Global Administrator, User Administrator, or License Administrator. And you now have a file to get onto the host, which means a Fleet custom package rather than a script-only one, and the "nothing to host" property of this whole approach is gone. Worth it for a handful of apps everyone needs, not for a self-service catalog.

## Wire it into GitOps

Register the package in the fleet where you want it available, in `fleets/<name>.yml` or `fleets/unassigned.yml`:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      categories:
        - "💻 Productivity"
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
      post_install_script:
        path: ../lib/windows/scripts/verify-company-portal.ps1
```

The label keeps the tile off hosts that can't use it, and it has to be defined in the `labels` section first:

```yaml
labels:
  - name: Windows
    query: "SELECT 1 FROM os_version WHERE platform = 'windows';"
    label_membership_type: dynamic
```

With `self_service: true`, the app appears on the end user's self-service page, reachable from the Fleet icon in the Windows system tray. Every app is three files and a YAML block, so adding or removing one is a pull request. Reviewers see exactly what will run, and the catalog is auditable and reversible. [Turn on GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) to make the Fleet UI reflect that these are code-managed.

Script-only packages don't support automatic install through a policy, and they don't take an `install_script` key, because the file's contents already are the install script. Self-service is the delivery mechanism here, which suits Store apps anyway.

## Generate the whole tile from a Store ID

Since only two lines differ between apps, the rest can be emitted. Fleet script-only packages are single files with no shared helper to import, so the boilerplate has to be inlined into each one, which is exactly the kind of work worth handing to a generator.

```powershell
# New-StoreAppTile.ps1 -StoreId 9WZDNCRFJ3PZ -DisplayName "Company Portal" -PackageName Microsoft.CompanyPortal
param(
    [Parameter(Mandatory)][string]$StoreId,
    [Parameter(Mandatory)][string]$DisplayName,
    [Parameter(Mandatory)][string]$PackageName,
    [string]$Category = "💻 Productivity"
)

$dir = "lib/windows/scripts"
New-Item -ItemType Directory -Path $dir -Force | Out-Null
$slug = $DisplayName.ToLower() -replace '[^a-z0-9]+', '-'

$body = @'
$exitCode = 0
$taskName = "fleet-store-$StoreId"
try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner | Select-Object -First 1).User
    if (-not $userName) { throw "No logged-on user, so there is no session to install into." }
    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $WingetArgs
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
            throw "Timed out waiting for winget to finish."
        }
    } while ($state -eq "Running" -or $state -eq "Queued")
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
}
exit $exitCode
'@

@"
`$StoreId = "$StoreId"
`$WingetArgs = "install --id `$StoreId --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity"
$body
"@ | Set-Content "$dir/install-$slug.ps1" -Encoding UTF8

@"
`$StoreId = "$StoreId"
`$WingetArgs = "uninstall --id `$StoreId --accept-source-agreements --disable-interactivity"
$body
"@ | Set-Content "$dir/uninstall-$slug.ps1" -Encoding UTF8

@"
`$PackageName = "$PackageName"
`$pkg = Get-AppxPackage -AllUsers -Name `$PackageName -ErrorAction SilentlyContinue
if (-not `$pkg) { Write-Host "`$PackageName is not installed for any user."; exit 1 }
Write-Host "Found `$(`$pkg[0].Name) `$(`$pkg[0].Version)."
exit 0
"@ | Set-Content "$dir/verify-$slug.ps1" -Encoding UTF8

@"

# Add to your fleet's software.packages:
    - path: ../$dir/install-$slug.ps1
      display_name: $DisplayName
      self_service: true
      categories:
        - "$Category"
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../$dir/uninstall-$slug.ps1
      post_install_script:
        path: ../$dir/verify-$slug.ps1
"@
```

Onboarding an app becomes three lookups and a command: get the Store ID from `winget search --source msstore`, get the MSIX package name from `Get-AppxPackage` after installing it once by hand, then run the generator and paste the printed block into your fleet file.

## The point

A Store app install is one winget command, and the reason this piece isn't one paragraph long is that Fleet runs as SYSTEM and the Microsoft Store only deals in users. That gap is a real architectural constraint, not an oversight waiting on a flag, so the honest answer is a wrapper that hands the work to the session where it belongs.

Write that wrapper once, verify against `Get-AppxPackage` instead of the registry, and the marginal cost of the next Store app is a twelve-character ID in a pull request.

## See it live

- Follow the [step-by-step guide](https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide) to build your first Store app tile end to end.
- Read the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for the full detail on script-only packages, pre-install queries, and uninstall scripts.
- Get a demo: [fleetdm.com/contact](https://fleetdm.com/contact).
- Join a free GitOps workshop: [fleetdm.com/workshops](https://fleetdm.com/workshops).

_Managing devices as code, one pull request at a time. Start with the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files) or [talk to us](https://fleetdm.com/contact)._

<meta name="articleTitle" value="Put Microsoft Store apps in Windows self-service with winget">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Use Fleet script-only .ps1 packages and winget to put Microsoft Store apps in Windows self-service, despite the SYSTEM context limit.">
