# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# The downloaded file is a bootstrapper that pulls the multi-GB payload during
# this step. --wait is required: without it the bootstrapper forks the real
# install to a background process and returns before it finishes.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

if (-not (Test-Path $exeFilePath)) {
  Write-Host "Error: Installer file not found at: $exeFilePath"
  Exit 1
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--quiet --wait --norestart"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

if ($exitCode -eq 1001 -or $exitCode -eq 1618) {
  Write-Host "Install failed: another Visual Studio Installer operation is already in progress (exit code $exitCode)"
  Exit 1
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
