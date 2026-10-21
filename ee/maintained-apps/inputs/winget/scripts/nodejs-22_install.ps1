$logFile = "${env:TEMP}/fleet-install-software.log"
$installerPath = "${env:INSTALLER_PATH}"

# Every Node.js MSI shares this upgrade code, and the installer refuses to run
# over a newer Node.js (WIX_DOWNGRADE_DETECTED -> 1603). This FMA pins Node 22,
# so remove any other Node.js first, then install.
$upgradeCode = '{47C07A3A-42EF-4213-A85D-8F5A59077C28}'
$targetVersion = '22.23.2'

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject WindowsInstaller.Installer
foreach ($code in @($installer.RelatedProducts($upgradeCode))) {
  $installedVersion = $installer.ProductInfo($code, 'VersionString')
  if ($installedVersion -eq $targetVersion) { continue }
  Write-Host "Removing Node.js $installedVersion ($code)"
  $removeProcess = Start-Process msiexec.exe -ArgumentList @("/quiet", "/norestart", "/x", $code) -PassThru -Wait
  if ($successCodes -notcontains $removeProcess.ExitCode) {
    Write-Host "Failed to remove Node.js $installedVersion (exit $($removeProcess.ExitCode))"
    Exit $removeProcess.ExitCode
  }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${installerPath}`"" `
  -PassThru -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
