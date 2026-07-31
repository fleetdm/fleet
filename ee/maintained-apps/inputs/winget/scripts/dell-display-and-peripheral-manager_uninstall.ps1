# Uninstalls Dell Display and Peripheral Manager (an InstallShield InstallScript
# MSI). The MSI ProductCode changes per release, so we look the product up in
# the registry by its exact DisplayName and uninstall via msiexec.

$softwareName = "Dell Display and Peripheral Manager"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareName) {
        $productCode = $key.PSChildName
        if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
            Write-Host "Unexpected uninstall key name (not a ProductCode GUID): $productCode"
            continue
        }
        Write-Host "Uninstalling product code: $productCode"
        $process = Start-Process -FilePath "msiexec.exe" `
            -ArgumentList "/x $productCode /qn /norestart" `
            -NoNewWindow -PassThru -Wait
        $exitCode = $process.ExitCode
        break
    }
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
# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
