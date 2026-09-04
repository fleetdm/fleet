# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# AnyDesk.exe is both the installer and the app: --install takes the target
# directory and registers the AnyDesk service, so it must run elevated.

$exeFilePath = "${env:INSTALLER_PATH}"
$installDir = Join-Path ${env:ProgramFiles(x86)} "AnyDesk"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--install `"$installDir`" --silent --update-auto --create-shortcuts --create-desktop-icon"
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
