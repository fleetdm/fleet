# Fleet uninstalls app by finding all related product codes for the specified upgrade code.
# GoToMeeting's winget manifest does not expose the UpgradeCode, so it is hard-coded
# here from the MSI's UpgradeCode property.
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product

foreach ($product_code in $inst.RelatedProducts('{BF62CF5F-6CB1-4010-8F05-F7EBB182D3EA}')) {
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

# All uninstalls succeeded; exit success
Exit 0
