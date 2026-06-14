# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Kiro ships as an Inno Setup-based installer (user scope, VS Code fork).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Inno Setup silent install. /mergetasks=!runcode prevents launching after install.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS /MERGETASKS=!runcode"
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
