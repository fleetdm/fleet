# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# Viscosity ships an Inno Setup installer. The base silent switches come from
# the winget InstallerSwitches (/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART).
# SparkLabs' own GPO/unattended deployment docs additionally require /NORUN so
# the installer does NOT launch Viscosity at the end -- with no interactive user
# (SYSTEM context) the launched app blocks the installer from exiting, which is
# what caused the headless install to hang. /NOCANCEL /NOCLOSEAPPLICATIONS
# /NORESTARTAPPLICATIONS prevent any remaining interactive prompts.
# Source: https://www.sparklabs.com/support/kb/article/deploy-viscosity-windows-under-a-gpo-group-policy-environment/

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NORUN /NOCANCEL /NOCLOSEAPPLICATIONS /NORESTARTAPPLICATIONS"
  PassThru = $true
  Wait = $true
}

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
