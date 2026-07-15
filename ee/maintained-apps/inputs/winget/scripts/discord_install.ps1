# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Discord ships a Squirrel-based installer. DiscordSetup.exe installs per-user to
# %LOCALAPPDATA%\Discord, writes an uninstall entry to HKCU, and then LAUNCHES
# Discord while the setup stub itself stays resident. Blocking on
# Start-Process -Wait therefore never returns (it hangs until the harness kills
# it), and the still-running DiscordSetup.exe keeps its own file locked, which
# breaks temp-directory cleanup. Instead: start it without waiting, poll for the
# install to complete, then stop the spawned processes so no handle is left
# holding the installer file open.

$installDir = Join-Path $env:LOCALAPPDATA "Discord"
$uninstallRoots = @(
    "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
)

function Test-DiscordInstalled {
    # Filesystem signal: Squirrel drops Update.exe and an app-<version> folder.
    if (Test-Path (Join-Path $installDir "Update.exe")) { return $true }
    if (Test-Path (Join-Path $installDir "app-*")) { return $true }
    # Registry signal: the uninstall entry the programs table (and our uninstall
    # script) key on. Discord registers per-user (HKCU) but check HKLM too.
    foreach ($root in $uninstallRoots) {
        $hit = Get-ChildItem $root -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
            Where-Object { $_.DisplayName -like "Discord*" }
        if ($hit) { return $true }
    }
    return $false
}

$installer = $null
try {
    $installer = Start-Process -FilePath "$exeFilePath" -ArgumentList "-s" -PassThru
} catch {
    Write-Host "Error starting installer: $_"
    Exit 1
}

# Poll well under the harness's 10-minute script timeout so the diagnostics
# below always get a chance to print on failure.
$deadline = (Get-Date).AddMinutes(7)
$installed = $false
while ((Get-Date) -lt $deadline) {
    if (Test-DiscordInstalled) { $installed = $true; break }
    Start-Sleep -Seconds 5
}

# Stop the auto-launched app and the installer stub so they don't keep file
# handles open (a running DiscordSetup.exe locks its own file, blocking cleanup).
if ($installer -and -not $installer.HasExited) {
    Stop-Process -Id $installer.Id -Force -ErrorAction SilentlyContinue
}
foreach ($name in @("Discord", "Update", "DiscordSetup")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

if ($installed) {
    Write-Host "Discord installed."
    Exit 0
}

# Nothing detected: surface what actually happened so the failure is actionable
# instead of a bare timeout.
Write-Host "Discord did not install within timeout."
if ($installer) {
    if ($installer.HasExited) {
        Write-Host "Installer process exit code: $($installer.ExitCode)"
    } else {
        Write-Host "Installer process was still running at timeout."
    }
}
Write-Host "Install dir '$installDir' exists: $(Test-Path $installDir)"
if (Test-Path $installDir) {
    Get-ChildItem $installDir -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty Name |
        ForEach-Object { Write-Host "  $_" }
}
foreach ($root in $uninstallRoots) {
    Get-ChildItem $root -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "*iscord*" } |
        ForEach-Object { Write-Host "Registry $root -> DisplayName='$($_.DisplayName)' Publisher='$($_.Publisher)'" }
}
Exit 1
