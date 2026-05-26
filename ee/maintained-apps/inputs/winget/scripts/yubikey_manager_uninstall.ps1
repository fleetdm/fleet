# Locates YubiKey Manager's NSIS uninstaller from the registry and runs it silently.

$displayName = "YubiKey Manager"
$publisher = "Yubico AB"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Best-effort: stop the GUI so the uninstaller does not fail on locked files.
Stop-Process -Name "yubikey-manager-qt" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "ykman" -Force -ErrorAction SilentlyContinue

$uninstallString = $uninstall.UninstallString
$exePath = ""
$arguments = ""

# Parse the uninstall string to extract executable path and existing arguments
# Handles both quoted and unquoted paths
if ($uninstallString -match '^"([^"]+)"(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} elseif ($uninstallString -match '^([^\s]+)(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

# Build argument list array, preserving existing arguments and adding /S for silent
# NSIS installers require /S flag for silent uninstall
$argumentList = @()
if ($arguments -ne '') {
    $argumentList += $arguments -split '\s+'
}
if ($argumentList -notcontains "/S" -and $arguments -notmatch '\b/S\b') {
    $argumentList += "/S"
}

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath = $exePath
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    if ($argumentList.Count -gt 0) {
        $processOptions.ArgumentList = $argumentList
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
