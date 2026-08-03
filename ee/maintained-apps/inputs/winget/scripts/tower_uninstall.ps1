# Fleet uninstalls app by finding all related product codes for the specified upgrade code.
# Tower's winget manifest declares a placeholder ProductCode ("MSI:Tower") instead of the
# real GUID, so the default product-code-based uninstall fails with 1619. The MSI's
# UpgradeCode is stable across releases, so uninstall via RelatedProducts instead.
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

foreach ($product_code in $inst.RelatedProducts('{871FD9D0-41D3-52BE-AF69-12F8B08740C0}')) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)

    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }

    # If the uninstall failed, bail
    if ($successCodes -notcontains $process.ExitCode) {
        Write-Output "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

# All uninstalls succeeded; exit success
Exit 0
