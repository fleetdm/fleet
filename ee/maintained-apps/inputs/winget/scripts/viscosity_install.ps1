# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# Viscosity ships an Inno Setup installer. Inno silent switches per the winget
# InstallerSwitches: /SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
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
