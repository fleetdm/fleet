# 1Password Uninstall Script
# Closes running processes before uninstalling to prevent hangs

# Each release has a new ProductCode, so uninstall whatever version is
# installed by resolving it from the stable UpgradeCode.
$upgradeCode = '{04B64D44-52BF-41A6-85BA-CDBA63AEB44D}'
$timeoutSeconds = 300  # 5 minute timeout per product
$successCodes = @(0, 3010, 1641)

# Close any running 1Password processes
Get-Process -Name "1Password*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$inst = New-Object -ComObject "WindowsInstaller.Installer"
$productCodes = @($inst.RelatedProducts($upgradeCode))
if ($productCodes.Count -eq 0) {
    Write-Output "No products found for upgrade code $upgradeCode"
    Exit 1
}

foreach ($product_code in $productCodes) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)

    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_INSTALL_FAILURE
    }

    Write-Output "Uninstall for $product_code exited $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) {
        Exit $process.ExitCode
    }
}

Exit 0
