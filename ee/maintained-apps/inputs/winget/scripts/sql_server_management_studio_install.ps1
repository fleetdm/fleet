# SSMS 22 ships as a Visual Studio bootstrapper (vs_SSMS.exe) that downloads and
# installs the real payload via the Visual Studio Installer. --wait makes the
# bootstrapper block until the install completes and return its real exit code.
$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010)

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--quiet --norestart --wait"
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
