# Fleet uninstalls app by finding all related product codes for the specified upgrade code
$inst = New-Object -ComObject "WindowsInstaller.Installer"
foreach ($product_code in $inst.RelatedProducts("$UPGRADE_CODE")) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -Wait -PassThru

    # If the uninstall failed, bail
    if ($process.ExitCode -ne 0) {
        Write-Output "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

# All uninstalls succeeded; exit success
Exit 0