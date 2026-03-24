# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$installDir = "C:\Program Files\Mozilla Firefox"
$maxWaitSeconds = 120

try {

# Start silent install without -Wait; the Firefox ESR installer launches the
# browser after installing and blocks until it is closed.
Start-Process -FilePath "$exeFilePath" -ArgumentList "/S"

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
