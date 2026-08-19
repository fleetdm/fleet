# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# JetBrains Toolbox ships only as a per-user installer (winget manifest has
# Scope: user). Per the winget InstallerSwitches, /headless runs it fully silent.
# This follows the same pattern as other user-scope FMAs (Granola, Discord, etc.).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/headless"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
