# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Streamlabs Desktop is an electron-builder (NSIS) installer; /S runs silent and
# /allusers forces the machine-wide install (default per-user otherwise).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S /allusers" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
