# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# digiSeal reader is a secrypt self-extracting installer; -silent runs it
# unattended and installs machine-wide (ARP written to HKLM).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "-silent"
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
