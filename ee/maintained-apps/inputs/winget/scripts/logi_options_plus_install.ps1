# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Logi Options+ silent install:
#   /quiet            - no UI
#   /analytics no     - decline the analytics opt-in prompt
# Logitech's installer reports success with exit code 0 or -1978335226
# (per the winget manifest's InstallerSuccessCodes).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/quiet /analytics no"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 0 -or $exitCode -eq -1978335226) {
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
