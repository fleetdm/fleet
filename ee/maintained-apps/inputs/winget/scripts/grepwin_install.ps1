# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# grepWin's MSI installs per user by default (ALLUSERS=2 + MSIINSTALLPERUSER=1),
# so under Fleet's SYSTEM context it lands in the SYSTEM account's profile,
# without a Start menu shortcut or context menu for anyone. Force a per-machine
# install into Program Files, and first remove a copy that an earlier install
# left in the SYSTEM profile.

$logFile = "${env:TEMP}/fleet-install-software.log"
$upgradeCode = '{C73ED533-953E-47E5-9A42-A48292BB7C6C}'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject "WindowsInstaller.Installer"

foreach ($productCode in @($installer.RelatedProducts($upgradeCode))) {
  # 0 = installed for this account only; 1 = installed machine-wide
  if ($installer.ProductInfo($productCode, "AssignmentType") -ne "0") { continue }

  Write-Host "Removing the copy of grepWin in the SYSTEM profile ($productCode)"
  $removeProcess = Start-Process msiexec.exe `
    -ArgumentList "/quiet /norestart /x $productCode" `
    -PassThru -Wait
  Write-Host "Removal exited $($removeProcess.ExitCode)"
  if ($successCodes -notcontains $removeProcess.ExitCode) {
    Exit $removeProcess.ExitCode
  }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" ALLUSERS=1 MSIINSTALLPERUSER=`"`" /i `"${env:INSTALLER_PATH}`"" `
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
