# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Discord ships a Squirrel-based installer. DiscordSetup.exe extracts the app to
# %LOCALAPPDATA%\Discord, writes an uninstall entry to HKCU, and then LAUNCHES
# Discord while the setup stub itself stays resident. Blocking on
# Start-Process -Wait therefore never returns (it hangs until the harness kills
# it), and the still-running DiscordSetup.exe keeps its own file locked, which
# breaks temp-directory cleanup.
#
# Poll for the uninstall REGISTRY entry, not the install directory: the app
# folder appears early during extraction, so keying on it would let us kill the
# installer before Squirrel finishes writing the registry entry that osquery's
# programs table -- and Fleet's inventory -- rely on. Once the entry exists the
# install is complete; then stop the spawned processes so no handle is left
# holding the installer file open.

$uninstallRoots = @(
    "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
)

function Get-DiscordUninstallEntry {
    foreach ($root in $uninstallRoots) {
        $hit = Get-ChildItem $root -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
            Where-Object { $_.DisplayName -like "Discord*" } |
            Select-Object -First 1
        if ($hit) { return $hit }
    }
    return $null
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
$entry = $null
while ((Get-Date) -lt $deadline) {
    $entry = Get-DiscordUninstallEntry
    if ($entry) { break }
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

if ($entry) {
    Write-Host "Discord registered: DisplayName='$($entry.DisplayName)' Version='$($entry.DisplayVersion)'"
    Exit 0
}

# Nothing detected: surface what actually happened so the failure is actionable.
Write-Host "Discord did not register within timeout."
if ($installer) {
    if ($installer.HasExited) {
        Write-Host "Installer process exit code: $($installer.ExitCode)"
    } else {
        Write-Host "Installer process was still running at timeout."
    }
}
$installDir = Join-Path $env:LOCALAPPDATA "Discord"
Write-Host "Install dir '$installDir' exists: $(Test-Path $installDir)"
if (Test-Path $installDir) {
    Get-ChildItem $installDir -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty Name |
        ForEach-Object { Write-Host "  $_" }
    # SquirrelSetup.log records exactly what the installer did, including
    # whether/where it wrote the uninstall registry entry and why it exited.
    $squirrelLog = Join-Path $installDir "SquirrelSetup.log"
    if (Test-Path $squirrelLog) {
        Write-Host "---- SquirrelSetup.log (tail) ----"
        Get-Content $squirrelLog -Tail 50 -ErrorAction SilentlyContinue |
            ForEach-Object { Write-Host $_ }
        Write-Host "---- end SquirrelSetup.log ----"
    }
}
foreach ($root in $uninstallRoots) {
    Get-ChildItem $root -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "*iscord*" } |
        ForEach-Object { Write-Host "Registry $root -> DisplayName='$($_.DisplayName)' Publisher='$($_.Publisher)' Version='$($_.DisplayVersion)'" }
}
Exit 1
