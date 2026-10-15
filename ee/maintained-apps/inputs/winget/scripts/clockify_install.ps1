# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Clockify's installer creates its shortcuts in the profile of the account that
# runs it, so a Fleet install would leave them in the Default user profile. This
# script adds a Start Menu shortcut for all users and removes those copies.

$logFile = "${env:TEMP}/fleet-install-software.log"
$msiFilePath = "${env:INSTALLER_PATH}"
$exePath = Join-Path $env:ProgramFiles "Clockify\ClockifyWindows.exe"
$timeoutSeconds = 600

try {

$process = Start-Process msiexec.exe `
  -ArgumentList "/i `"$msiFilePath`" /quiet /norestart /lv `"$logFile`"" `
  -PassThru
# Reading Handle now keeps ExitCode available after WaitForExit(timeout).
$null = $process.Handle
if ($process.WaitForExit($timeoutSeconds * 1000)) {
  $exitCode = $process.ExitCode
} else {
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Write-Host "Timed out after $timeoutSeconds seconds waiting for the installer."
  $exitCode = 1460  # ERROR_TIMEOUT
}
Write-Host "Install exit code: $exitCode"

# MSI reboot-required success codes.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { $exitCode = 0 }

if ($exitCode -ne 0) {
  Get-Content $logFile -Tail 500
  Exit $exitCode
}

if (-not (Test-Path -LiteralPath $exePath)) {
  Throw "Clockify installed, but $exePath was not found."
}

$shell = New-Object -ComObject WScript.Shell
$startMenuShortcut = Join-Path $env:ProgramData "Microsoft\Windows\Start Menu\Programs\Clockify.lnk"
$shortcut = $shell.CreateShortcut($startMenuShortcut)
$shortcut.TargetPath = $exePath
$shortcut.WorkingDirectory = Split-Path $exePath -Parent
$shortcut.Save()
Write-Host "Created $startMenuShortcut"

$defaultProfile = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList').Default
foreach ($stray in @(
  (Join-Path $defaultProfile "AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Clockify.lnk"),
  (Join-Path $defaultProfile "Desktop\Clockify.lnk")
)) {
  if ((Test-Path -LiteralPath $stray) -and $shell.CreateShortcut($stray).TargetPath -eq $exePath) {
    Remove-Item -LiteralPath $stray -Force -ErrorAction SilentlyContinue
  }
}

Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
