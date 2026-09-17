$logFile = "${env:TEMP}/fleet-install-software.log"

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

# The installer's own .NET Framework check runs out of order during silent
# installs and always fails, so this script performs the same check (.NET
# Framework 4.8.1, release 533320 or later) and passes the result to the
# installer via WIX_IS_NETFRAMEWORK_481_OR_LATER_INSTALLED.

try {

$release = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full' -ErrorAction SilentlyContinue).Release
if (-not $release -or $release -lt 533320) {
  Write-Host "This application requires .NET Framework 4.8.1 or later. Please install .NET Framework 4.8.1, then retry."
  Exit 1
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${env:INSTALLER_PATH}`" WIX_IS_NETFRAMEWORK_481_OR_LATER_INSTALLED=1" `
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
