# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# NordLayer ships an InstallShield/burn-style EXE wrapping an MSI
$processOptions = @{
  FilePath     = "$exeFilePath"
  ArgumentList = "/exenoui /quiet /norestart"
  PassThru     = $true
  Wait         = $true
  NoNewWindow  = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
