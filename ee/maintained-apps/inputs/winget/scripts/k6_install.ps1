# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# k6's MSI doesn't set ALLUSERS, so without ALLUSERS=1 it registers only for the
# account that runs it (SYSTEM, under Fleet). A copy registered that way by an
# earlier install is removed first so it isn't left behind next to the new one.

$logFile = "${env:TEMP}/fleet-install-software.log"
$upgradeCode = '{8C5F9495-F697-4780-ABFD-90DCC2F681B6}'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject "WindowsInstaller.Installer"

foreach ($productCode in @($installer.RelatedProducts($upgradeCode))) {
  # 0 = installed for this account only; 1 = installed machine-wide
  if ($installer.ProductInfo($productCode, "AssignmentType") -ne "0") { continue }

  Write-Host "Removing the copy of k6 registered for the SYSTEM account ($productCode)"
  $removeProcess = Start-Process msiexec.exe `
    -ArgumentList "/quiet /norestart /x $productCode" `
    -PassThru -Wait
  if ($successCodes -notcontains $removeProcess.ExitCode) {
    Write-Host "Removal exited $($removeProcess.ExitCode)"
    Exit $removeProcess.ExitCode
  }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" ALLUSERS=1 /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
