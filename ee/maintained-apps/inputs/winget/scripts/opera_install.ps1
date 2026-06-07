# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Opera's standalone (Nullsoft) installer. A bare /silent still LAUNCHES the
# browser after install, which leaves a foreground process running and hangs a
# headless SYSTEM install. Disable every interactive/launch behavior.
# Switches verified against the winget InstallerSwitches (/silent + Custom
# /allusers=1) and silentinstallhq.com, which recommends
# "/silent /allusers=1 /setdefaultbrowser=0 /launchbrowser=0".
#   /install            - install mode (vs. update/uninstall)
#   /silent             - no UI
#   /allusers=1         - machine-wide install (visible to SYSTEM context)
#   /launchbrowser=0    - do not launch the browser after install (current builds)
#   /launchopera=0      - same, switch name used by older builds; harmless if ignored
#   /setdefaultbrowser=0- do not prompt/attempt to set default browser
#   /pintotaskbar=0     - no taskbar pin (needs an interactive desktop)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/install /silent /allusers=1 /launchbrowser=0 /launchopera=0 /setdefaultbrowser=0 /pintotaskbar=0"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install requires a reboot (exit code $exitCode); treating as success."
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
