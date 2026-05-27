# BlueJ ships a per-user WiX MSI. Fleet runs installs elevated (SYSTEM), so we
# pass ALLUSERS=2 — the same switch winget uses — which resolves to a per-machine
# install (into %ProgramFiles%\BlueJ) when run elevated.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`" ALLUSERS=2" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
