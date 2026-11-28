# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# Webex's MSI is a dual-purpose package that defaults to a per-user install
# (ALLUSERS=2, MSIINSTALLPERUSER=1), so under Fleet's SYSTEM context it lands in
# the SYSTEM account's profile, where no one can open it. Force the per-machine
# install into Program Files, which is also what enables the Outlook
# integration. First remove a copy an earlier install left in the SYSTEM
# profile, so it isn't left behind unreachable.

$logFile = "${env:TEMP}/fleet-install-software.log"
$upgradeCode = '{B9DCA8E9-B3A9-419A-9D6F-9BC3557EE72C}'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject "WindowsInstaller.Installer"

foreach ($productCode in @($installer.RelatedProducts($upgradeCode))) {
  # 0 = installed for this account only; 1 = installed machine-wide
  if ($installer.ProductInfo($productCode, "AssignmentType") -ne "0") { continue }

  Write-Host "Removing the copy of Webex in the SYSTEM profile ($productCode)"
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
