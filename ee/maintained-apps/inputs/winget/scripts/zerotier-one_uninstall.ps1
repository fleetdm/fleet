# Each ZeroTier One release has a new ProductCode, so this script finds every
# installed version by ZeroTier One's UpgradeCode and uninstalls each one.
$ErrorActionPreference = 'Stop'
$upgradeCode = '{B0E2A5F3-88B6-4E77-B922-CB4739B4C4C8}'
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

foreach ($product_code in $inst.RelatedProducts($upgradeCode)) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

    $completed = $process.WaitForExit($timeoutSeconds * 1000)

    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }

    if ($successCodes -notcontains $process.ExitCode) {
        Write-Output "Uninstall for $($product_code) exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
}

Exit 0
