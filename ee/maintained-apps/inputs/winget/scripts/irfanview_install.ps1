# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# IrfanView uses its own installer; /silent runs it unattended and /allusers=1
# installs machine-wide. /desktop and /group control shortcut creation.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/silent /desktop=1 /group=1 /allusers=1 /assoc=0" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
