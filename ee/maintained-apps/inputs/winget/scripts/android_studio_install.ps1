$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1223)

try {

# While studio.exe or studio64.exe runs (in any session), the installer waits
# on a message box asking to close it. Fleet runs it as SYSTEM, where nobody
# can dismiss that box, so the install would hang. Stop Android Studio first.
# Unsaved changes are lost.
$studioNames = @("studio", "studio64")
$running = Get-Process -Name $studioNames -ErrorAction SilentlyContinue
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
if (Get-Process -Name $studioNames -ErrorAction SilentlyContinue) {
    Write-Host "Error: Android Studio is still running. The installer would wait for it to close."
    Exit 1
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
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
