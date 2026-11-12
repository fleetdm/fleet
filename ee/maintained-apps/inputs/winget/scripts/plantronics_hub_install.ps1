$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010, 1223)

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/install /quiet /norestart"
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
