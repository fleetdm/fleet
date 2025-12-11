# Attempts to locate 7-Zip's product code and uninstall it using msiexec
# This script is version-agnostic and will find any installed version of 7-Zip

$displayNamePattern = "7-Zip*"
$publisher = "Igor Pavlov"
$upgradeCode = "{23170F69-40C1-2702-0000-000004000000}"

# Close any running 7-Zip processes before uninstalling
Write-Host "Closing any running 7-Zip processes..."
Get-Process -Name "7z*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Get-Process -Name "7zFM" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$productCodes = @()
$timeoutSeconds = 300  # 5 minute timeout per product

# Method 1: Use upgrade code to find all related products (most reliable for MSI)
Write-Host "Searching for 7-Zip using upgrade code: $upgradeCode"
try {
  $installer = New-Object -ComObject WindowsInstaller.Installer
  $relatedProducts = $installer.RelatedProducts($upgradeCode)
  foreach ($productCode in $relatedProducts) {
    $productCodes += $productCode
    Write-Host "Found product code via upgrade code: $productCode"
  }
} catch {
  Write-Host "Upgrade code method failed: $_"
}

# Method 2: Search registry for any 7-Zip installation
if ($productCodes.Count -eq 0) {
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
      $foundCode = ($items | Select-Object -First 1).PSChildName
      if ($foundCode -notin $productCodes) {
        $productCodes += $foundCode
        Write-Host "Found product code in registry: $foundCode"
      }
    }
  }
}

# Method 3: Use WindowsInstaller COM object to find by name
if ($productCodes.Count -eq 0) {
  Write-Host "Trying WindowsInstaller COM object method..."
  try {
    if (-not $installer) {
      $installer = New-Object -ComObject WindowsInstaller.Installer
    }
    $products = $installer.Products
    
    foreach ($product in $products) {
      try {
        $productName = $installer.ProductInfo($product, "ProductName")
        if ($productName -like $displayNamePattern) {
          if ($product -notin $productCodes) {
            $productCodes += $product
            Write-Host "Found product code via WindowsInstaller: $product (Product: $productName)"
          }
        }
      } catch {
        # Continue searching
      }
    }
  } catch {
    Write-Host "WindowsInstaller COM object method failed: $_"
  }
}

if ($productCodes.Count -eq 0) {
  Write-Host "Product code not found for 7-Zip"
  Exit 0  # Not found = success for Fleet
}

# Uninstall all found product codes
$allSucceeded = $true
foreach ($productCode in $productCodes) {
  Write-Host "Attempting to uninstall product code: $productCode"
  
  try {
    # Use /qn (quiet no UI) for better compatibility
    # REBOOT=ReallySuppress prevents any reboot prompts
    $process = Start-Process msiexec -ArgumentList @("/x", $productCode, "/qn", "/norestart", "REBOOT=ReallySuppress") -PassThru -NoNewWindow
    
    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    if (-not $completed) {
      Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
      Write-Host "Uninstall timed out after $timeoutSeconds seconds for product code: $productCode"
      $allSucceeded = $false
      continue
    }
    
    # Check exit code
    if ($process.ExitCode -eq 0) {
      Write-Host "Uninstall successful for product code: $productCode (exit code: 0)"
    } else {
      Write-Host "Uninstall failed for product code: $productCode with exit code: $($process.ExitCode)"
      $allSucceeded = $false
    }
  } catch {
    Write-Host "Error running uninstaller for product code $productCode : $_"
    $allSucceeded = $false
  }
}

# Add a delay to ensure filesystem and registry are updated
if ($allSucceeded) {
  Write-Host "All uninstalls completed successfully"
  Start-Sleep -Seconds 5
  Exit 0
} else {
  Write-Host "One or more uninstalls failed"
  Exit 1603  # ERROR_UNINSTALL_FAILURE
}
