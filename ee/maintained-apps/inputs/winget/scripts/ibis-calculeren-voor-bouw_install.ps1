# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Ibis Calculeren voor Bouw ships as an InstallShield wrapper around an MSI.
# /S runs the wrapper silently; /V passes /quiet /norestart through to msiexec
# for a silent machine-wide install.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/S /V`"/quiet /norestart`"" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
