$exeFilePath = "${env:INSTALLER_PATH}"

try {

# EnterpriseDB (BitRock InstallBuilder) installer. Switches from the winget
# manifest: --mode unattended for a non-interactive install, --unattendedmodeui
# none to suppress all progress UI. Installs per-machine under
# C:\Program Files\PostgreSQL\17 and registers in HKLM Add/Remove Programs.

# The EDB uninstaller keeps the data directory. Installing over that leftover
# cluster when no PostgreSQL 17 install is registered fails as SYSTEM: the
# installer grants "<computer name>$" access to it, which icacls can't resolve.
# Move it aside so the install creates a new cluster; the old data is kept.
$dataDir = Join-Path $env:ProgramFiles "PostgreSQL\17\data"
$registered = Test-Path "HKLM:\SOFTWARE\PostgreSQL\Installations\postgresql-x64-17"
if (-not $registered -and (Get-ChildItem $dataDir -Force -ErrorAction SilentlyContinue | Select-Object -First 1)) {
  $backupDir = "$dataDir.$(Get-Date -Format 'yyyyMMddHHmmss')"
  Move-Item -Path $dataDir -Destination $backupDir -ErrorAction Stop
  Write-Host "Moved data directory left by a previous install to $backupDir"
}

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
