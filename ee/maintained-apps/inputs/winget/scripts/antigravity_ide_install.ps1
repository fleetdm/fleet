$exeFilePath = "${env:INSTALLER_PATH}"

$ExpectedExitCodes = @(0, 3010)

try {

# Antigravity uses an Inno Setup-based installer (user scope).
# /MERGETASKS=!runcode deselects the "launch after install" task.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS /MERGETASKS=!runcode"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# Stop the app if the installer auto-launched it despite !runcode
Stop-Process -Name "Antigravity" -Force -ErrorAction SilentlyContinue

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
