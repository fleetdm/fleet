$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010, 1223)

try {

# Git for Windows uses an Inno Setup installer. These are the switches documented
# in the winget manifest (/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART);
# /NORESTARTAPPLICATIONS /CLOSEAPPLICATIONS avoid reboot prompts on locked files.
# Fleet runs elevated (SYSTEM), so the installer installs machine-wide (all users).
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NORESTARTAPPLICATIONS /CLOSEAPPLICATIONS"
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
