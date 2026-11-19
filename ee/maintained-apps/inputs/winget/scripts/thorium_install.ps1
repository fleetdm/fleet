# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# /allusers installs Thorium Reader machine-wide. Without it, the installer puts
# the app in the SYSTEM account's profile, where no one can open it.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
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
