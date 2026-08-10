# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Vendor-documented silent switches (winget InstallerSwitches: Silent +
# Custom): /silent runs without dialogs/prompts, /noreboot suppresses the
# reboot prompt, and /AutoUpdateCheck=disabled keeps Citrix's own updater
# from taking over version management.

try {

$installProcess = Start-Process -FilePath "${env:INSTALLER_PATH}" `
  -ArgumentList "/silent /noreboot /AutoUpdateCheck=disabled" `
  -PassThru -Wait

$exitCode = $installProcess.ExitCode
Write-Host "Install exit code: $exitCode"

# Treat msiexec-style reboot-required codes as success too.
if ($exitCode -eq 0 -or $exitCode -eq 3010 -or $exitCode -eq 1641) {
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
