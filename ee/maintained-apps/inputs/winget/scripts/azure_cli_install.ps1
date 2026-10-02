# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Installing over an existing Azure CLI of the same version turns into a repair,
# which Windows Installer's SecureRepair blocks and fails with 1603. Any existing
# copy is removed first so this is always a clean install. Azure CLI extensions
# and config live in the user profile and are not affected.

$msiFilePath = "${env:INSTALLER_PATH}"
$logFile = "${env:TEMP}/fleet-install-software.log"
$softwareName = "Microsoft Azure CLI (64-bit)"
$softwarePublisher = "Microsoft Corporation"

# 3010 = ERROR_SUCCESS_REBOOT_REQUIRED, 1641 = ERROR_SUCCESS_REBOOT_INITIATED.
$successCodes = @(0, 3010, 1641)

try {

$machineKey       = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

[array]$existing = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath } |
    Where-Object { $_.DisplayName -eq $softwareName -and $_.Publisher -eq $softwarePublisher }

foreach ($entry in $existing) {
    $productCode = $entry.PSChildName
    if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
        Write-Host "Skipping unexpected uninstall key name (not a ProductCode GUID): $productCode"
        continue
    }
    Write-Host "Removing installed $softwareName $($entry.DisplayVersion) (product code $productCode)."
    $removeProcess = Start-Process msiexec.exe `
        -ArgumentList "/x $productCode /qn /norestart" -NoNewWindow -PassThru -Wait
    Write-Host "Remove exit code: $($removeProcess.ExitCode)"
    if (-not ($successCodes -contains $removeProcess.ExitCode)) {
        Exit $removeProcess.ExitCode
    }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${msiFilePath}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

Write-Host "Install exit code: $($installProcess.ExitCode)"
if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
