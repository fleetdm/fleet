# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Vernier Spectral Analysis ships as a Nullsoft (NSIS) installer; /S runs silent.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
  PassThru = $true
  Wait = $true
  NoNewWindow = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if (-not (Test-Path $exeFilePath)) {
  Write-Host "Installer not found at $exeFilePath"
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
