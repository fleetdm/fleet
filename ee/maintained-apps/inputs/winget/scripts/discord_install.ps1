# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add argument to install silently
# Discord uses Squirrel installer (common for Electron apps)
# Discord uses -s (lowercase) for silent installation
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "-s"
  PassThru = $true
}

# Start process and track exit code. Don't block indefinitely: the Squirrel
# setup has been observed to hang without exiting. Bound the wait, and on
# timeout kill the installer tree and surface the Squirrel logs.
$process = Start-Process @processOptions
$timedOut = $false
if (-not $process.WaitForExit(8 * 60 * 1000)) {
  $timedOut = $true
  Write-Host "Installer did not exit within 8 minutes; killing process tree."
  & taskkill /PID $process.Id /T /F | Out-Null
  $process.WaitForExit(30 * 1000) | Out-Null
}
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"

# On failure, surface the Squirrel setup logs so the error is visible in
# Fleet's script output instead of just a bare exit code.
if ($timedOut -or $exitCode -ne 0) {
  foreach ($log in @("$env:LOCALAPPDATA\SquirrelTemp\SquirrelSetup.log", "$env:LOCALAPPDATA\Discord\SquirrelSetup.log")) {
    if (Test-Path $log) {
      Write-Host "--- $log (last 50 lines) ---"
      Get-Content $log -Tail 50 | ForEach-Object { Write-Host $_ }
    }
  }
}

if ($timedOut) { Exit 1 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
