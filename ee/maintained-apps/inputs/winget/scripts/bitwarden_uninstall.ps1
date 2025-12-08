# Attempts to locate Bitwarden's uninstaller from registry and execute it silently

$displayName = "Bitwarden"
$publisher = "8bit Solutions LLC"
$productCode = "{173a9bac-6f0d-50c4-8202-4744c69d091a}"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# Function to check if Bitwarden is still installed
function Test-BitwardenInstalled {
    foreach ($p in $paths) {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            ($_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)) -or
            ($_.PSChildName -eq $productCode) -or
            ($_.ProductCode -eq $productCode)
        }
        if ($items) { return $true }
    }
    return $false
}

# Kill any running Bitwarden processes before uninstalling
Stop-Process -Name "Bitwarden" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    ($_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)) -or
    ($_.PSChildName -eq $productCode) -or
    ($_.ProductCode -eq $productCode)
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

# Build argument list array, preserving existing arguments and adding /S for silent
$argumentList = @()
if ($arguments -ne '') {
    # Split existing arguments and add them
    $argumentList += $arguments -split '\s+'
}
$argumentList += "/S"

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

$exitCode = 0
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
    
    # Wait for registry to update after uninstall
    Start-Sleep -Seconds 5
    
    # Check for any running uninstaller processes
    $uninstallerProcesses = Get-Process | Where-Object { $_.ProcessName -like "*uninstall*" -or $_.ProcessName -like "*Bitwarden*" }
    if ($uninstallerProcesses) {
        Write-Host "Waiting for uninstaller processes to complete..."
        $timeout = 30
        $elapsed = 0
        while ($elapsed -lt $timeout -and (Get-Process | Where-Object { $_.ProcessName -like "*uninstall*" -or $_.ProcessName -like "*Bitwarden*" })) {
            Start-Sleep -Seconds 2
            $elapsed += 2
        }
    }
    
    # Retry checking if app is still installed (up to 10 times, 2 seconds apart)
    $maxRetries = 10
    $retryCount = 0
    while ($retryCount -lt $maxRetries -and (Test-BitwardenInstalled)) {
        Write-Host "App still detected, waiting for uninstall to complete... (attempt $($retryCount + 1)/$maxRetries)"
        Start-Sleep -Seconds 2
        $retryCount++
    }
    
    if (Test-BitwardenInstalled) {
        Write-Host "Warning: App still detected after uninstall, but uninstaller exited with code $exitCode"
        # Don't fail if uninstaller reported success
        if ($exitCode -eq 0) {
            Write-Host "Uninstaller reported success, treating as successful"
            $exitCode = 0
        }
    } else {
        Write-Host "App successfully removed from registry"
    }
    
} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

Exit $exitCode

