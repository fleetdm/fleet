# SSMS 22 ships as a Visual Studio bootstrapper (vs_SSMS.exe) that downloads and
# installs the real payload via the Visual Studio Installer. --wait makes the
# bootstrapper block until the install completes and return its real exit code.
$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010)

try {

# The bootstrapper exits 1 while SSMS is open, because the Visual Studio
# Installer can't update files the IDE holds. Fleet runs as SYSTEM, outside the
# user's session, so it can't ask SSMS to close: stop it first. Unsaved queries
# are lost.
$getRunning = { @(Get-Process -Name "SSMS" -ErrorAction SilentlyContinue) }
$running = & $getRunning
foreach ($proc in $running) {
  Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
  Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Wait-Process reports a timeout as a non-terminating error, so confirm the IDE
# really stopped rather than letting the bootstrapper fail on its open files.
$stillRunning = & $getRunning
if ($stillRunning.Count) {
  Write-Host "Error: SQL Server Management Studio is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Quit it and try again."
  Exit 1
}

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
