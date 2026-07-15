# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Discord ships a Squirrel-based installer. DiscordSetup.exe installs per-user to
# %LOCALAPPDATA%\Discord, writes an uninstall entry to HKCU, and then LAUNCHES
# Discord while the setup stub itself stays resident. Blocking on
# Start-Process -Wait therefore never returns (it hangs until the harness kills
# it), and the still-running DiscordSetup.exe keeps its own file locked, which
# breaks temp-directory cleanup. Instead: start it without waiting, poll the
# HKCU uninstall key until the install registers, then stop the spawned
# processes so no handles are left holding the installer file open.

$displayName = "Discord"
$publisher = "Discord Inc."
$uninstallKey = "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
$deadline = (Get-Date).AddMinutes(5)
$registered = $false

try {
    Start-Process -FilePath "$exeFilePath" -ArgumentList "-s" | Out-Null

    while ((Get-Date) -lt $deadline) {
        $entry = Get-ChildItem $uninstallKey -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
            Where-Object { $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher }
        if ($entry) {
            $registered = $true
            break
        }
        Start-Sleep -Seconds 5
    }
} catch {
    Write-Host "Error: $_"
}

# Stop the auto-launched app and the installer stub so they don't keep file
# handles open (a running DiscordSetup.exe locks its own file, blocking cleanup).
foreach ($name in @("Discord", "Update", "DiscordSetup")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

if ($registered) {
    Write-Host "Discord registered in HKCU."
    Exit 0
}

Write-Host "Discord did not register within timeout."
Exit 1
