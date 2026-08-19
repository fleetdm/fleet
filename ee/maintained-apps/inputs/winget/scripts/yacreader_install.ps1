$exeFilePath = "${env:INSTALLER_PATH}"

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
  PassThru = $true
  Wait = $true
  NoNewWindow = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if (-not (Test-Path $exeFilePath)) {
  Write-Host "Installer not found at $exeFilePath"
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode
