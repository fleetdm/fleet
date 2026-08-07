# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# The downloaded file is a bootstrapper (~4 MB), not the IDE itself - it
# downloads the multi-GB payload from Microsoft during this step, so install
# time depends on the host's network speed. --wait is required: without it
# the bootstrapper forks the real install to a background process and
# returns almost immediately, well before the install finishes.

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

# 3010/1641: install succeeded but a reboot is pending/was triggered. Fleet
# treats any nonzero exit as a failed install, so map these to success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

# 1001/1618: another Visual Studio Installer operation is already running.
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
