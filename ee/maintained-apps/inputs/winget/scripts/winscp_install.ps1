# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# WinSCP uses an Inno Setup installer. /ALLUSERS forces a machine-wide install
# (the default is per-user, which would land in the SYSTEM profile when Fleet
# runs as SYSTEM). /VERYSILENT /SUPPRESSMSGBOXES /NORESTART are the standard
# Inno Setup silent switches.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/ALLUSERS /VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
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
