# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# KeePass uses an Inno Setup installer: /ALLUSERS forces a machine-wide install
# (the default is per-user, which would land in the SYSTEM profile when Fleet
# runs as SYSTEM). Setup refuses to run while KeePass is open, so any running
# instance is stopped first -- unsaved database changes are lost.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

Get-Process -Name "KeePass" -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "Stopping running KeePass process (PID $($_.Id))."
  Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
}

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
