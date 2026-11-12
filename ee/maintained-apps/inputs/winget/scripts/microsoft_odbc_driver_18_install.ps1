# The MSI refuses to install without IACCEPTMSODBCSQLLICENSETERMS=YES, which the
# default MSI install script does not pass.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`" IACCEPTMSODBCSQLLICENSETERMS=YES" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($installProcess.ExitCode -eq 3010 -or $installProcess.ExitCode -eq 1641) { Exit 0 }

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
