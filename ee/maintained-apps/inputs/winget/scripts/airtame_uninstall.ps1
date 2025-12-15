# Fleet uninstalls Airtame MSI
# Tries product code first, then falls back to upgrade code lookup
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product
$upgradeCode = "{E29931C0-80A4-45BE-8F54-BDC39F9D3C63}"
$productCode = "{AA404F77-BC09-4A42-A93F-EBB102306039}"

function Test-ProductInstalled {
    param([string]$ProductCode)
    try {
        $product = $inst.Products.Item($ProductCode)
        return $true
    } catch {
        return $false
    }
}

# Function to uninstall by product code and verify it's gone
function Uninstall-ProductCode {
    param([string]$Code)
    
    Write-Output "Attempting to uninstall product code: $Code"
    
    # Check if product is installed first
    if (-not (Test-ProductInstalled -ProductCode $Code)) {
        Write-Output "Product code $Code is not installed"
        return $true
    }
    
    # Uninstall using /qn (quiet, no UI) instead of /quiet
    $process = Start-Process msiexec -ArgumentList @("/qn", "/x", $Code, "/norestart") -PassThru -Wait -NoNewWindow
    
    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    
    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Output "Uninstall timed out for $Code"
        return $false
    }
    
    # Check exit code
    if ($process.ExitCode -ne 0) {
        Write-Output "Uninstall for $Code exited with code $($process.ExitCode)"
        return $false
    }
    
    # Wait a moment for Windows Installer to update
    Start-Sleep -Seconds 2
    
    # Verify product is actually gone
    $stillInstalled = Test-ProductInstalled -ProductCode $Code
    if ($stillInstalled) {
        Write-Output "Product $Code is still installed after uninstall attempt"
        return $false
    }
    
    Write-Output "Successfully uninstalled $Code"
    return $true
}

# First, try uninstalling by product code directly
$uninstalled = Uninstall-ProductCode -Code $productCode

# If that didn't work, try finding all products with the upgrade code
if (-not $uninstalled) {
    Write-Output "Product code uninstall failed, trying upgrade code lookup..."
    $relatedProducts = $inst.RelatedProducts($upgradeCode)
    
    if ($relatedProducts.Count -eq 0) {
        Write-Output "No products found with upgrade code $upgradeCode"
        Exit 1
    }
    
    foreach ($relatedProductCode in $relatedProducts) {
        Write-Output "Found related product: $relatedProductCode"
        $result = Uninstall-ProductCode -Code $relatedProductCode
        if (-not $result) {
            Write-Output "Failed to uninstall $relatedProductCode"
            Exit 1
        }
    }
}

# Final verification - check if any products with this upgrade code still exist
$remainingProducts = $inst.RelatedProducts($upgradeCode)
if ($remainingProducts.Count -gt 0) {
    Write-Output "Warning: $($remainingProducts.Count) product(s) still found with upgrade code after uninstall"
    Exit 1
}

# Also verify the specific product code is gone
if (Test-ProductInstalled -ProductCode $productCode) {
    Write-Output "Product code $productCode is still installed"
    Exit 1
}

Write-Output "Uninstall completed successfully"
Exit 0
