# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 240

try {

# Silent install flags from winget manifest for Mozilla.Firefox.ESR (nullsoft installer).
$installArgs = "/S /PreventRebootRequired=true"

# In CI/headless environments, disable the Maintenance Service to prevent hangs.
# On real devices, the Maintenance Service is kept enabled for seamless background updates.
if ($env:CI -eq "true") {
    $installArgs += " /MaintenanceService=false"
    Write-Host "CI environment detected, disabling MaintenanceService"
}

Write-Host "Install args: $installArgs"
$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList $installArgs `
  -PassThru

Write-Host "Installer started (PID: $($process.Id)), waiting up to $timeoutSeconds seconds..."

$completed = $process.WaitForExit($timeoutSeconds * 1000)

if (-not $completed) {
    Write-Host "Installation timed out after $timeoutSeconds seconds, killing process..."
    $process.Kill()
    Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# Wait briefly for any child processes to finish and release file locks
Start-Sleep -Seconds 5

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
