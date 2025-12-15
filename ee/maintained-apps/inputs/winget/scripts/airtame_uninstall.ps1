# Fleet uninstalls app by finding all related product codes for the specified upgrade code
# Falls back to product code if upgrade code lookup finds nothing
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product
$upgradeCode = "{E29931C0-80A4-45BE-8F54-BDC39F9D3C63}"
$productCode = "{AA404F77-BC09-4A42-A93F-EBB102306039}"

# First, try to find and uninstall products using the upgrade code
$relatedProducts = $inst.RelatedProducts($upgradeCode)
$foundProducts = $false

foreach ($product_code in $relatedProducts) {
    $foundProducts = $true
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru
    
    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    
    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }
    
    # If the uninstall failed, bail
    if ($process.ExitCode -ne 0) {
        Write-Output "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

# If no products were found with upgrade code, try product code directly
if (-not $foundProducts) {
    Write-Output "No products found with upgrade code, trying product code directly..."
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $productCode, "/norestart") -PassThru
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    
    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }
    
    if ($process.ExitCode -ne 0) {
        Write-Output "Uninstall by product code exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

# All uninstalls succeeded; exit success
Exit 0
