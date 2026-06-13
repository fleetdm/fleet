# Attempts to locate Binance's NSIS uninstaller from the registry and run it silently.
# The registry DisplayName includes the version (e.g. "Binance 2.1.0"), so match a prefix.

$displayName = "Binance"
$publisher = "BinanceTech"

# Check HKCU first (per-user installs), then HKLM as fallback.
$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and $_.DisplayName -like "$displayName*" -and ($publisher -eq "" -or $_.Publisher -like "$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallString = $uninstall.UninstallString
$exePath = ""
$arguments = ""
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

$argumentList = @()
if ($arguments -ne '') { $argumentList += $arguments -split '\s+' }
if ($argumentList -notcontains "/S" -and $arguments -notmatch '\b/S\b') { $argumentList += "/S" }

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath = $exePath
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }
    if ($argumentList.Count -gt 0) { $processOptions.ArgumentList = $argumentList }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
