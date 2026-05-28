# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Evernote 10.x+ NSIS installer. Per silentinstallhq's Evernote guide the
# documented silent machine-wide flags are "/AllUsers /S" in that exact order
# and casing — using "/S /allusers" caused an access violation (0xc0000005)
# in CI on Evernote 11.x. Honor the documented form.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/AllUsers /S"
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
