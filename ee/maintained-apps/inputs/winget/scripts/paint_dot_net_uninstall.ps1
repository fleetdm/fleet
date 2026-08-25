# The ProductCode changes every release and differs between the .exe and .msi
# variants; the UpgradeCode is stable across both.
$upgradeCode = '{04A40F40-A207-4B48-AED7-6AA532E43275}'
$timeoutSeconds = 300
$successCodes = @(0, 3010, 1641)

try {
    $inst = New-Object -ComObject "WindowsInstaller.Installer"
    # An empty list means nothing to remove; a failed query means we don't know.
    $productCodes = @()
    try {
        $productCodes = @($inst.RelatedProducts($upgradeCode))
    } catch {
        Write-Host "Could not query related products for upgrade code $upgradeCode. Error: $_"
        Exit 1
    }

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
