$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S", "/allusers"
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
