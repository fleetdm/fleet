# Uninstall for GoToMeeting.
#
# The winget installer is the GoToMeeting "Setup" bootstrapper (ARPSYSTEMCOMPONENT=1,
# so it hides itself from Programs and Features). It installs the actual GoToMeeting
# app, which self-registers a separate, visible uninstall entry (DisplayName like
# "GoToMeeting <version>") whose uninstaller is G2MUninstall.exe.
#
# We locate that entry and run G2MUninstall.exe directly.

$softwareNameLike = "GoToMeeting*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike) {
        $selected = $key
        break
    }
}

if (-not $selected) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

# Best-effort: stop running GoToMeeting processes so the uninstaller doesn't
# fail on locked files.
foreach ($proc in @("g2mstart", "g2mlauncher", "g2mcomm", "g2muicore", "g2mupdate", "GoToMeeting", "GoTo")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

# Extract just the uninstaller exe path from whichever registry string is
# available; we supply the silent switches ourselves.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

if (-not $uninstallCommand) {
    Write-Host "Selected entry has no UninstallString: $($selected.DisplayName)"
    Exit 1
}

$exePath = ""
if ($uninstallCommand -match '^\s*"([^"]+)"') {
    # Quoted path
    $exePath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
    # Unquoted path that may contain spaces (e.g. "C:\Program Files (x86)\...")
    $exePath = $matches[1]
} else {
    Throw "Could not parse uninstaller path from: $uninstallCommand"
}

# Vendor-documented silent uninstall switches. /ForAllUsers matches the
# machine-wide install (G2MINSTALLFORALLUSERS=1); /silent is the correct silent
# switch (NOT /S, which G2MUninstall.exe ignores).
$uninstallArgs = "/uninstall /ForAllUsers /silent"

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $uninstallArgs"

$process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# Treat msiexec-style reboot-required success codes as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
