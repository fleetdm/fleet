# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Check for any existing Firefox processes before install
$firefoxProcs = Get-Process -Name "firefox" -ErrorAction SilentlyContinue
if ($firefoxProcs) {
    Write-Host "Found running Firefox processes, stopping them..."
    $firefoxProcs | Stop-Process -Force
    Start-Sleep -Seconds 2
}

# Use Mozilla's -ms flag for silent enterprise installation
# https://firefox-source-docs.mozilla.org/browser/installer/windows/installer/FullConfig.html
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "-ms"
  PassThru = $true
  Wait = $true
}

Write-Host "Starting Firefox ESR install with -ms flag..."
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
