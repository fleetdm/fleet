# The x86 and x64 redistributables share a DisplayName and differ only by
# product code, so resolve the installed product from the x64 upgrade code.
$upgradeCode = '{00160000-00D1-0000-1000-0000000FF1CE}'
$timeoutSeconds = 300
$successCodes = @(0, 3010, 1641)

try {
    $inst = New-Object -ComObject "WindowsInstaller.Installer"
    $productCodes = @()
    try { $productCodes = @($inst.RelatedProducts($upgradeCode)) } catch { }

    if ($productCodes.Count -eq 0) { Write-Host "No installed product found for upgrade code $upgradeCode."; Exit 0 }

    foreach ($productCode in $productCodes) {
        $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $productCode, "/norestart") -PassThru
        if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Write-Host "Uninstall for $productCode timed out."
            Exit 1603
        }
        Write-Host "Uninstall for $productCode exited $($process.ExitCode)"
        if ($successCodes -notcontains $process.ExitCode) { Exit $process.ExitCode }
    }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit 0
