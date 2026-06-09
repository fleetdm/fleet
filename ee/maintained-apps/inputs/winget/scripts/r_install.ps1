$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010, 1223)

try {

# R for Windows uses an Inno Setup installer. /VERYSILENT installs with no UI,
# /SUPPRESSMSGBOXES suppresses prompts, /NORESTART prevents reboots. Fleet runs
# elevated (SYSTEM), so the installer installs machine-wide.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
