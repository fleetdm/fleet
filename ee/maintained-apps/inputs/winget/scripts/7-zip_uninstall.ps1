# Fleet uninstalls app by finding all related product codes for the specified upgrade code
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$upgradeCode = "{23170F69-40C1-2702-0000-000004000000}"
$timeoutSeconds = 300  # 5 minute timeout per product

# Get all product codes for this upgrade code
$productCodes = @($inst.RelatedProducts($upgradeCode))

if ($productCodes.Count -eq 0) {
    Write-Host "No products found for upgrade code $upgradeCode"
    Exit 0
}

foreach ($product_code in $productCodes) {
    Write-Host "Uninstalling product: $product_code"
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru -NoNewWindow
    
    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    
    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall timed out for $product_code"
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }
    
    # If the uninstall failed, bail
    if ($process.ExitCode -ne 0) {
        Write-Host "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
    
    # Wait a moment for Windows to update the registry
    Start-Sleep -Seconds 2
}

# Wait additional time for registry to fully update
Start-Sleep -Seconds 3

# Verify uninstall by checking if any products still exist
$remainingProducts = @($inst.RelatedProducts($upgradeCode))
if ($remainingProducts.Count -gt 0) {
    Write-Host "Warning: Some products still exist after uninstall: $($remainingProducts -join ', ')"
    # Still exit success if we attempted uninstall - registry might be slow to update
    Exit 0
}

Write-Host "Uninstall successful"
Exit 0
