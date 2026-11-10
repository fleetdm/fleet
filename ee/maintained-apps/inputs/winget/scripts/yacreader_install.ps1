$exeFilePath = "${env:INSTALLER_PATH}"

# Setup fails with exit code 5 when YACReader's files are in use. Fleet runs as
# SYSTEM, outside the user's session, so it can't ask the app to close: stop it
# first. Unsaved changes are lost.
$running = Get-Process -ErrorAction SilentlyContinue |
  Where-Object { $_.Path -like "*\YACReader\*" }
foreach ($proc in $running) {
  Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
  Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
  PassThru = $true
  Wait = $true
  NoNewWindow = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# A file can stay locked for a moment after the app is stopped. Setup rolls
# back cleanly on exit code 5, so retry once.
if ($exitCode -eq 5 -and $running) {
  Write-Host "Install exit code: 5. Retrying in 10 seconds."
  Start-Sleep -Seconds 10
  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
}

if (-not (Test-Path $exeFilePath)) {
  Write-Host "Installer not found at $exeFilePath"
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode
