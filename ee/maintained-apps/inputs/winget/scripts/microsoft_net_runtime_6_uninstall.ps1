# Attempts to locate Microsoft .NET Runtime 6's uninstaller from registry and execute it silently

$displayName = "Microsoft .NET Runtime"
$publisher = "Microsoft Corporation"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -like "*$displayName*") -and ($_.DisplayName -like "*6.0*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
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

# Build argument list array, preserving existing arguments and adding silent flag
$argumentList = @()
if ($arguments -ne '') {
    # Split existing arguments and add them
    $argumentList += $arguments -split '\s+'
}

# Detect if this is an MSI uninstall (MsiExec.exe) - MSI uses /quiet, not /silent
$isMsi = $exePath -like "*msiexec.exe" -or $exePath -like "*MsiExec.exe"

if ($isMsi) {
    # MSI uninstalls use /quiet and /norestart
    if ($argumentList -notcontains "/quiet") {
        $argumentList += "/quiet"
    }
    if ($argumentList -notcontains "/norestart") {
        $argumentList += "/norestart"
    }
} else {
    # Non-MSI installers typically use /silent
    if ($argumentList -notcontains "/silent" -and $argumentList -notcontains "--silent") {
        $argumentList += "/silent"
    }
}

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath = $exePath
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
    Write-Host "Error: $_"
    Exit 1
}

