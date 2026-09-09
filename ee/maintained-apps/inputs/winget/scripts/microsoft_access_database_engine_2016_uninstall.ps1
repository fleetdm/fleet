# The x86 and x64 redistributables share a DisplayName and differ only by
# product code, so resolve the installed product from the x64 upgrade code.
$upgradeCode = '{00160000-00D1-0000-1000-0000000FF1CE}'
$displayName = 'Microsoft Access database engine 2016 (English)'
$timeoutSeconds = 300
$successCodes = @(0, 3010, 1641)

# The x64 build registers here; the x86 build registers under Wow6432Node.
$nativeUninstallKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

try {
    $inst = New-Object -ComObject "WindowsInstaller.Installer"
    $productCodes = @()
    try {
        $productCodes = @($inst.RelatedProducts($upgradeCode))
    } catch {
        # RelatedProducts throws when nothing is installed for the upgrade code,
        # so confirm against the registry before calling that a clean uninstall.
        Write-Host "Could not enumerate products for upgrade code ${upgradeCode}: $($_.Exception.Message)"
        $registered = Get-ItemProperty -Path $nativeUninstallKey -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -eq $displayName } |
            Select-Object -First 1
        if ($registered) {
            Write-Host "'$displayName' is still registered, so this is a real failure."
            Exit 1
        }
    }

    if ($productCodes.Count -eq 0) { Write-Host "No installed product found for upgrade code $upgradeCode."; Exit 0 }

    foreach ($productCode in $productCodes) {
        $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $productCode, "/norestart") -PassThru
        # Keeps .ExitCode readable after the process ends.
        $null = $process.Handle
        if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Write-Host "Uninstall for $productCode timed out."
            Exit 1603
        }
        $exitCode = $process.ExitCode
        if ($null -eq $exitCode) {
            Write-Host "Uninstall for $productCode reported no exit code."
            Exit 1603
        }
        Write-Host "Uninstall for $productCode exited $exitCode"
        if ($successCodes -notcontains $exitCode) { Exit $exitCode }
    }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit 0
