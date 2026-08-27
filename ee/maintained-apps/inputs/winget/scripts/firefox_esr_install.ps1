# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$installDir = "C:\Program Files\Mozilla Firefox"
$maxWaitSeconds = 120

try {

# Start silent install without -Wait; the Firefox ESR installer launches the
# browser after installing and blocks until it is closed.
# /RegisterDefaultAgent=false skips registering Mozilla's Default Browser Agent
# scheduled task, which can't be registered from the SYSTEM context this script
# runs in: the attempt fails with 0x80070534 ("No mapping between account names
# and security IDs") in the Application event log and can surface a firefox.exe
# error dialog on the user's first launch.
Start-Process -FilePath "$exeFilePath" -ArgumentList "/S /RegisterDefaultAgent=false"

# Poll for installation to complete
$elapsed = 0
while ($elapsed -lt $maxWaitSeconds) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    if (Test-Path "$installDir\firefox.exe") {
        Write-Host "Firefox ESR installed successfully after $elapsed seconds"
        Exit 0
    }
}

Write-Host "Timed out waiting for Firefox ESR to install"
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
