$exeFilePath = "${env:INSTALLER_PATH}"

try {

# EnterpriseDB (BitRock InstallBuilder) installer. Switches from the winget
# manifest: --mode unattended for a non-interactive install, --unattendedmodeui
# none to suppress all progress UI. Installs per-machine under
# C:\Program Files\PostgreSQL\15 and registers in HKLM Add/Remove Programs.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--mode", "unattended", "--unattendedmodeui", "none"
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
