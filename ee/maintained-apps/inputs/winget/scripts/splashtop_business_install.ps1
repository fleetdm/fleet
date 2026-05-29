# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Splashtop Business ships an EXE wrapper around an MSI. The bootstrapper
# accepts the same flag form as Splashtop Streamer (same vendor, same wrapper):
# `prevercheck /s /i <MSI properties>`. Splashtop Business additionally needs
# CA_UPGRADE=1 (documented in the winget manifest's Silent string) to enable
# upgrade-in-place when reinstalling.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "prevercheck /s /i CA_UPGRADE=1 hidewindow=1"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
if ($exitCode -eq 3010) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
