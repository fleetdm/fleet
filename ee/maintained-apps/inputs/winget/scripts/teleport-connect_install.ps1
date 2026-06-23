# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# Teleport Connect ships an NSIS (electron-builder) installer.
# NSIS installers require /S for silent installation.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
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
