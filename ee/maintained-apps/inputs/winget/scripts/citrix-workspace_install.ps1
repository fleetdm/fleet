# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Citrix Workspace app silent, machine-wide install.
# Switches verified against the winget InstallerSwitches (/silent /norestart),
# silentinstallhq.com, and Citrix deployment docs.
#   /silent              - unattended install
#   /noreboot            - never reboot (synonym /norestart; vendor docs use /noreboot)
#   /AutoUpdateCheck=Disabled - prevent the auto-update component from reaching out
#                          to the network/prompting, which can otherwise block a
#                          headless SYSTEM install.
#   /EnableCEIP=false    - skip the Customer Experience Improvement prompt/telemetry.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/silent /noreboot /AutoUpdateCheck=Disabled /EnableCEIP=false"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# Treat reboot-required codes as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install requires a reboot (exit code $exitCode); treating as success."
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
