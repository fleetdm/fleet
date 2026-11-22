# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# Wrike's MSI only installs per machine when run with a UI, so a silent install
# under Fleet's SYSTEM context lands in the SYSTEM account's profile, where no
# one can open it. Force a per-machine install into Program Files. First remove
# a copy that an earlier install left in the SYSTEM profile: while it exists,
# the MSI switches back to a per-user install into that copy's folder.

$logFile = "${env:TEMP}/fleet-install-software.log"
$upgradeCode = '{FD36F414-EF8A-48CB-8D20-BD321B03D375}'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject "WindowsInstaller.Installer"

foreach ($productCode in @($installer.RelatedProducts($upgradeCode))) {
  # 0 = installed for this account only; 1 = installed machine-wide
  if ($installer.ProductInfo($productCode, "AssignmentType") -ne "0") { continue }

  Write-Host "Removing the copy of Wrike in the SYSTEM profile ($productCode)"
  $removeProcess = Start-Process msiexec.exe `
    -ArgumentList "/quiet /norestart /x $productCode" `
    -PassThru -Wait
  if ($successCodes -notcontains $removeProcess.ExitCode) {
    Write-Host "Removal exited $($removeProcess.ExitCode)"
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
