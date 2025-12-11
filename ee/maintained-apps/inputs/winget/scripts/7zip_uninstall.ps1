# Attempts to locate 7-Zip's product code and uninstall it using msiexec
# This script is version-agnostic and will find any installed version of 7-Zip

$displayNamePattern = "7-Zip*"
$publisher = "Igor Pavlov"

# Close any running 7-Zip processes before uninstalling
Write-Host "Closing any running 7-Zip processes..."
Get-Process -Name "7z*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Get-Process -Name "7zFM" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# First, try using $PACKAGE_ID if it's available and is a valid product code (GUID)
$productCode = $null
if ($PACKAGE_ID -and $PACKAGE_ID -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$') {
  Write-Host "Using product code from PACKAGE_ID: $PACKAGE_ID"
  $productCode = $PACKAGE_ID
}

# If PACKAGE_ID didn't work or wasn't available, search registry for any 7-Zip installation
# This matches any version like "7-Zip 25.01 (x64)" or "7-Zip 24.07 (x64)"
if (-not $productCode) {
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

# Also try using WindowsInstaller COM object to find by name (more reliable for MSI)
if (-not $productCode) {
  Write-Host "Trying WindowsInstaller COM object method..."
  try {
    $installer = New-Object -ComObject WindowsInstaller.Installer
    $products = $installer.Products
    
    foreach ($product in $products) {
      try {
        $productName = $installer.ProductInfo($product, "ProductName")
        if ($productName -like $displayNamePattern) {
          $productCode = $product
          Write-Host "Found product code via WindowsInstaller: $productCode (Product: $productName)"
          break
        }
      } catch {
        # Continue searching
      }
    }
  } catch {
    Write-Host "WindowsInstaller COM object method failed: $_"
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
  # Use /qn (quiet no UI) instead of /quiet for better compatibility
  # REBOOT=ReallySuppress prevents any reboot prompts
  $process = Start-Process msiexec -ArgumentList @("/x", $productCode, "/qn", "/norestart", "REBOOT=ReallySuppress") -PassThru -NoNewWindow
  
  # Wait for process with timeout
  $completed = $process.WaitForExit($timeoutSeconds * 1000)
  if (-not $completed) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Write-Host "Uninstall timed out after $timeoutSeconds seconds"
    Exit 1603  # ERROR_UNINSTALL_FAILURE
  }
  
  # Check exit code and output result
  if ($process.ExitCode -eq 0) {
    Write-Host "Uninstall successful (exit code: 0)"
    # Add a delay to ensure filesystem and registry are updated
    Start-Sleep -Seconds 3
    Exit 0
  } else {
    Write-Host "Uninstall failed with exit code: $($process.ExitCode)"
    Exit $process.ExitCode
  }
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
