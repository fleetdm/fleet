# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# AnyDesk uses its own installer with documented silent flags from the
# winget manifest: --install "C:\Program Files (x86)\AnyDesk" --silent.

$exeFilePath = "${env:INSTALLER_PATH}"
$installDir = "C:\Program Files (x86)\AnyDesk"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--install `"$installDir`" --silent --update-auto --create-desktop-icon --create-shortcuts"
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
