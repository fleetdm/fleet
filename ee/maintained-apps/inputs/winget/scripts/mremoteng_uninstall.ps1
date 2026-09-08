# Uninstalls mRemoteNG. The MSI product code changes per build, so the product
# is looked up in the registry by name and publisher rather than pinned to the
# build this app shipped.

$softwareName = "mRemoteNG"
$softwarePublisher = "Next Generation Software"

$machineKey       = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# 0 = success, 3010 = reboot required, 1641 = reboot initiated.
$successCodes = @(0, 3010, 1641)

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -ne $softwareName -or $key.Publisher -ne $softwarePublisher) { continue }

    $productCode = $key.PSChildName
    if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
        Write-Host "Unexpected uninstall key name (not a ProductCode GUID): $productCode"
        continue
    }

    Write-Host "Uninstalling $softwareName $($key.DisplayVersion) (product code $productCode)."
    $process = Start-Process -FilePath "msiexec.exe" `
        -ArgumentList "/x $productCode /qn /norestart" `
        -NoNewWindow -PassThru -Wait
    $exitCode = $process.ExitCode
    break
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($null -eq $exitCode) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
if ($successCodes -contains $exitCode) { Exit 0 }
Exit $exitCode
