# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# dotTrace is installed by a web bootstrapper that downloads the product at install time.
# /PerMachine=True installs for all users instead of into the SYSTEM profile, and
# /VsVersion=* adds the Visual Studio integration to every VS instance present.
# The standalone profiler installs whether or not Visual Studio is present.

$exeFilePath = "${env:INSTALLER_PATH}"

$registryTimeoutSeconds = 3300

$uninstallPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-DotTraceEntries {
  Get-ChildItem -Path $uninstallPaths -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object {
      $_.DisplayName -like 'JetBrains dotTrace*' -and
      $_.Publisher -like '*JetBrains*'
    }
}

try {

$logFile = Join-Path $env:TEMP 'dottrace-install.log'

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/Silent=True /PerMachine=True /SpecificProductNames=dotTrace /VsVersion=* /LogFile=`"$logFile`""
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Installer exit code: $exitCode"

$deadline = (Get-Date).AddSeconds($registryTimeoutSeconds)
$entries = @(Get-DotTraceEntries)
while ($entries.Count -eq 0 -and (Get-Date) -lt $deadline) {
  $installerWasRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  Start-Sleep -Seconds 10
  $entries = @(Get-DotTraceEntries)
  $installerIsRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  if ($entries.Count -eq 0 -and -not $installerWasRunning -and -not $installerIsRunning) { break }
}

if ($entries.Count -eq 0) {
  Write-Host "The dotTrace uninstall registry entry did not appear."
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
