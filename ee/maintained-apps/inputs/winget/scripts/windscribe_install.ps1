# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Windscribe's installer takes exactly these switches: -silent hides its
# interface and -no-auto-start stops it from launching Windscribe when it
# finishes. Any unrecognized switch makes it exit with an error.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "-silent", "-no-auto-start"
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
