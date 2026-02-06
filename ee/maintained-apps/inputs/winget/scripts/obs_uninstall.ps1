# Attempts to locate OBS Studio's uninstaller from registry and execute it silently

$displayName = "OBS Studio"
$publisher = "OBS Project"

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

# Kill any running OBS processes before uninstalling
Stop-Process -Name "obs64" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "obs32" -Force -ErrorAction SilentlyContinue

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
    # Split existing arguments and add them
    $argumentList += $arguments -split '\s+'
}
# Append /S if not already present
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

