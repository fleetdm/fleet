# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# DBeaver ships as a Nullsoft (NSIS) installer. /S runs silent, /allusers
# installs machine-wide per the winget manifest's machine-scope switch.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /allusers"
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
