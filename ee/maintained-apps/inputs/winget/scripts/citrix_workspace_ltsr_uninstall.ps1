# Uninstall for Citrix Workspace App LTSR.
#
# The installer self-registers under Programs and Features as "Citrix
# Workspace <version>" (e.g. "Citrix Workspace 2507"), with an UninstallString
# that points at the vendor's own uninstaller (TrolleyExpress.exe on older
# builds, CWAInstaller.exe on 2311.1+). We look up that entry, extract just
# the executable path, and re-run it with the vendor-documented silent
# uninstall switches rather than trusting whatever args are already in the
# registry string.

$softwareNameLike = "Citrix Workspace *"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
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

# Vendor-documented silent uninstall switches.
$uninstallArgs = "/uninstall /cleanup /silent"

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $uninstallArgs"

$process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# Treat msiexec-style reboot-required codes as success too.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
