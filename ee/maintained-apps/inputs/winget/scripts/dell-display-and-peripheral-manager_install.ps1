# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# DDPM's installer aborts (exit 0x80042000) in a non-interactive SYSTEM session
# unless told it is headless. Dell's documented managed-deployment switches:
#   /Silent            - no UI
#   /HeadlessMode=true - required for SYSTEM/session-0 (no desktop) installs
#   /TelemetryConsent=false - decline telemetry (no consent prompt)
#   /TurnOffCA         - disable the app's own auto-update (Fleet manages updates)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/Silent /HeadlessMode=true /TelemetryConsent=false /TurnOffCA"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
