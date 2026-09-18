# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# dotPeek is installed by the JetBrains dotUltimate web installer, which downloads
# the product at install time. /PerMachine=True keeps it out of the SYSTEM profile
# and /VsVersion=0 selects the standalone (non-Visual Studio) product.
# https://resharper-support.jetbrains.com/hc/en-us/articles/207241485

$exeFilePath = "${env:INSTALLER_PATH}"

$registryTimeoutSeconds = 3300

$uninstallPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-DotPeekEntries {
  Get-ChildItem -Path $uninstallPaths -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object {
      $_.DisplayName -like 'JetBrains dotPeek*' -and
      $_.Publisher -like '*JetBrains*'
    }
}

try {

$logFile = Join-Path $env:TEMP 'dotpeek-install.log'

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/Silent=True /PerMachine=True /SkipEtwService=True /VsVersion=0 /LogFile=`"$logFile`""
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Installer exit code: $exitCode"

$deadline = (Get-Date).AddSeconds($registryTimeoutSeconds)
$entries = @(Get-DotPeekEntries)
while ($entries.Count -eq 0 -and (Get-Date) -lt $deadline) {
  $installerWasRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  Start-Sleep -Seconds 10
  $entries = @(Get-DotPeekEntries)
  $installerIsRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  if ($entries.Count -eq 0 -and -not $installerWasRunning -and -not $installerIsRunning) { break }
}

if ($entries.Count -eq 0) {
  Write-Host "The dotPeek uninstall registry entry did not appear."
  if (Test-Path $logFile) {
    Write-Host "--- last 50 lines of $logFile ---"
    Get-Content $logFile -Tail 50 | ForEach-Object { Write-Host $_ }
  }
  if ($exitCode -eq 0) { Exit 1 }
  Exit $exitCode
}

foreach ($entry in $entries) {
  Write-Host "Installed: DisplayName='$($entry.DisplayName)' DisplayVersion='$($entry.DisplayVersion)' Publisher='$($entry.Publisher)'"
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
