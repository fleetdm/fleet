$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--silent --skip_launch_new --gsuite_shortcuts=false"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

# Setup exits 1610 while a previous install or uninstall (for example, the
# Google Drive driver removal) is waiting on a restart.
if ($exitCode -eq 1610) {
  Write-Host "Install exit code: $exitCode (a previous Google Drive install or uninstall is waiting on a restart; restart the host, then install again)"
  Exit $exitCode
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
