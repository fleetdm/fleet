$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$uninstallKeys = @(
    "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKCU:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)

$app = Get-ItemProperty $uninstallKeys -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -like "DeepL*" } |
    Select-Object -First 1

if (-not $app) {
    Write-Host "DeepL not found in registry; nothing to uninstall."
    exit 0
}

# Prefer QuietUninstallString verbatim — 0install already includes --batch --background there.
$useQuiet = $false
$cmd = $null
$arguments = ""

if ($app.QuietUninstallString) {
    $uninstallString = $app.QuietUninstallString
    $useQuiet = $true
} else {
    $uninstallString = $app.UninstallString
}

if (-not $uninstallString) {
    Write-Host "No uninstall string found for DeepL."
    exit 1
}

# Parse the uninstall string into executable + arguments.
if ($uninstallString -match '^"([^"]+)"\s*(.*)$') {
    # Quoted executable path.
    $cmd = $matches[1]
    $arguments = $matches[2].Trim()
} elseif ($uninstallString -match '^(\S+\.exe)\s*(.*)$') {
    # Unquoted single-token executable.
    $cmd = $matches[1]
    $arguments = $matches[2].Trim()
} else {
    # Bare path with no arguments.
    $cmd = $uninstallString.Trim()
    $arguments = ""
}

# Only append 0install silent flags when we fell back to the plain UninstallString.
if (-not $useQuiet) {
    if ($arguments -notmatch '--batch')      { $arguments = ("$arguments --batch").Trim() }
    if ($arguments -notmatch '--background') { $arguments = ("$arguments --background").Trim() }
}

Write-Host "Uninstalling DeepL: $cmd $arguments"

if ($arguments) {
    $proc = Start-Process -FilePath $cmd -ArgumentList $arguments -Wait -PassThru -NoNewWindow
} else {
    $proc = Start-Process -FilePath $cmd -Wait -PassThru -NoNewWindow
}

$exitCode = $proc.ExitCode
Write-Host "Uninstaller exited with code $exitCode"

# 0install / common success codes.
if ($exitCode -eq 0 -or $exitCode -eq 3010 -or $exitCode -eq 1605) {
    exit 0
}

exit $exitCode
