# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Wacom Tablet Driver ships as an InstallShield-style EXE; /s is the documented
# silent flag per the winget manifest.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/s"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
# Wacom's InstallShield-based installer returns exit code 2 to signal
# "reboot required" (non-standard — most installers use 3010). The driver
# is fully installed and visible in Add/Remove Programs at that point.
# Treat 2 and 3010 as success.
if ($exitCode -eq 2 -or $exitCode -eq 3010) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
