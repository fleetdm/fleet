# Uninstall Egnyte Desktop App by finding all related product codes for its
# upgrade code. Mirrors the generated upgrade-code uninstall, but treats the MSI
# reboot codes as success:
#   0    = success
#   3010 = success, reboot required
#   1641 = success, reboot initiated
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product

foreach ($product_code in $inst.RelatedProducts('{03909D9B-F5F2-41F3-ABF9-4FCE077F028D}')) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)

    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }

    # Bail only on a genuine failure; 0/3010/1641 are MSI success codes.
    if ($process.ExitCode -ne 0 -and $process.ExitCode -ne 3010 -and $process.ExitCode -ne 1641) {
        Write-Output "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

# All uninstalls succeeded; exit success
Exit 0
