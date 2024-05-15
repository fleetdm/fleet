$logFile = "${env:TEMP}/fleet-remove-software.log"

$removeProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /x `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

exit $removeProcess.ExitCode
