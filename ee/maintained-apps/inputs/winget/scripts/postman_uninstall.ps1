# Attempts to locate Postman's uninstaller from registry and execute it silently

$displayName = "Postman"
$publisher = "Postman Inc."

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Kill any running Postman processes before uninstalling
Stop-Process -Name "Postman" -Force -ErrorAction SilentlyContinue

# Prefer QuietUninstallString if available, otherwise use UninstallString
$uninstallString = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} elseif ($uninstall.UninstallString) {
    $uninstall.UninstallString
} else {
    Write-Host "No uninstall string found"
    Exit 0
}

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

# If QuietUninstallString was used, it likely already has silent args - use as-is
# Otherwise, add --silent to UninstallString
$useQuietString = $uninstall.QuietUninstallString -ne $null

if ($useQuietString) {
    # Use QuietUninstallString exactly as it appears (it should already be silent)
    Write-Host "Using QuietUninstallString as-is"
    Write-Host "Uninstall executable: $exePath"
    Write-Host "Uninstall arguments: $arguments"
    
    try {
        $processOptions = @{
            FilePath = $exePath
            NoNewWindow = $true
            PassThru = $true
            Wait = $true
        }
        
        if ($arguments -ne '') {
            $processOptions.ArgumentList = $arguments -split '\s+'
        }
        
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        
        Write-Host "Uninstall exit code: $exitCode"
    } catch {
        Write-Host "Error running uninstaller: $_"
        $exitCode = 1
    }
} else {
    # Build argument list array, adding --silent for silent uninstall
    $argumentList = @()
    if ($arguments -ne '') {
        # Split existing arguments and add them
        $argumentList += $arguments -split '\s+'
    }
    
    # Only add --silent if it's not already present
    if ($uninstallString -notmatch '--silent' -and $arguments -notmatch '--silent') {
        $argumentList += "--silent"
    }
    
    Write-Host "Uninstall executable: $exePath"
    Write-Host "Uninstall arguments: $($argumentList -join ' ')"

    try {
        $processOptions = @{
            FilePath = $exePath
            ArgumentList = $argumentList
            NoNewWindow = $true
            PassThru = $true
            Wait = $true
        }
        
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        
        Write-Host "Uninstall exit code: $exitCode"
    } catch {
        Write-Host "Error running uninstaller: $_"
        $exitCode = 1
    }
}

# Wait for uninstaller to complete and registry to update
# Postman's uninstaller may be asynchronous, so wait and poll
$maxWaitSeconds = 30
$waitIntervalSeconds = 1
$waitedSeconds = 0

while ($waitedSeconds -lt $maxWaitSeconds) {
    Start-Sleep -Seconds $waitIntervalSeconds
    $waitedSeconds += $waitIntervalSeconds
    
    # Check if Postman is still in registry
    $stillInstalled = $false
    foreach ($p in $paths) {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
        }
        if ($items) {
            $stillInstalled = $true
            break
        }
    }
    
    if (-not $stillInstalled) {
        Write-Host "Postman successfully uninstalled after $waitedSeconds seconds"
        $exitCode = 0
        break
    }
}

if ($stillInstalled) {
    Write-Host "Warning: Postman still appears to be installed after $maxWaitSeconds seconds"
    # Don't fail - exit code from uninstaller process takes precedence
}

Exit $exitCode
