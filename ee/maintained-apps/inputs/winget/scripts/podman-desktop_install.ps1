# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Podman Desktop ships as an electron-builder NSIS ("nullsoft") installer.
# "/S" runs it silently; "/ALLUSERS" forces the per-machine install so the ARP
# entry lands under HKLM (Fleet installs run as SYSTEM).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S", "/ALLUSERS"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
