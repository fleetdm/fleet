# Attempts to locate 7-Zip's product code and uninstall it using msiexec
# This script is version-agnostic and will find any installed version of 7-Zip

$displayNamePattern = "7-Zip*"
$publisher = "Igor Pavlov"

# First, try using $PACKAGE_ID if it's available and is a valid product code (GUID)
$productCode = $null
if ($PACKAGE_ID -and $PACKAGE_ID -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$') {
  Write-Host "Using product code from PACKAGE_ID: $PACKAGE_ID"
  $productCode = $PACKAGE_ID
} else {
  # Fall back to searching registry for any 7-Zip installation
  # This matches any version like "7-Zip 25.01 (x64)" or "7-Zip 24.07 (x64)"
  Write-Host "Searching registry for 7-Zip installation..."
  
  $paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
  )

  foreach ($p in $paths) {
    $items = Get-ChildItem -Path $p -ErrorAction SilentlyContinue | ForEach-Object {
      Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
    } | Where-Object {
      $_.DisplayName -and $_.DisplayName -like $displayNamePattern -and ($publisher -eq "" -or $_.Publisher -eq $publisher) -and $_.PSChildName -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$'
    }
    if ($items) {
      $productCode = ($items | Select-Object -First 1).PSChildName
      Write-Host "Found product code in registry: $productCode"
      break
    }
  }
}

if (-not $productCode) {
  Write-Host "Product code not found for 7-Zip"
  Exit 0  # Not found = success for Fleet
}

Write-Host "Found product code: $productCode"
Write-Host "Attempting to uninstall using msiexec..."

$timeoutSeconds = 300  # 5 minute timeout

try {
  $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $productCode, "/norestart") -PassThru -NoNewWindow
  
  # Wait for process with timeout
  $completed = $process.WaitForExit($timeoutSeconds * 1000)
  
  if (-not $completed) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Write-Host "Uninstall timed out after $timeoutSeconds seconds"
    Exit 1603  # ERROR_UNINSTALL_FAILURE
  }
  
  # Check exit code and output result
  if ($process.ExitCode -eq 0) {
    Write-Host "Uninstall successful"
    # Add a small delay to ensure filesystem is updated
    Start-Sleep -Seconds 2
    Exit 0
  } else {
    Write-Host "Uninstall failed with exit code: $($process.ExitCode)"
    Exit $process.ExitCode
  }
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
