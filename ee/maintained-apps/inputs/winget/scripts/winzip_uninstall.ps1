# Fleet uninstalls app by finding all related product codes for the specified upgrade code
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

foreach ($product_code in $inst.RelatedProducts('{CD95F661-A5C4-44F5-A6AA-ECDD91C240A3}')) {
    # SHOW_WEBPAGE=0 stops WinZip's uninstaller from opening its feedback page in a browser.
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "SHOW_WEBPAGE=0", "/norestart") -PassThru

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
