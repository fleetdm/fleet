# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# NoSQL Workbench uses an NSIS (nullsoft) installer; /allusers for machine scope
$processOptions = @{
  FilePath     = "$exeFilePath"
  ArgumentList = "/S /allusers"
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
