# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# RStudio ships an NSIS installer built with NsisMultiUser, which rejects /S on
# its own with exit code 666660 (invalid command-line parameters). /allusers
# picks machine scope, matching the winget manifest's machine-scope switch.

$exeFilePath = "${env:INSTALLER_PATH}"

# Setup exits 2 without installing while any rstudio.exe is running, in any
# session. Fleet runs as SYSTEM, outside the user's session, so it can't ask
# RStudio to close: stop it first, along with the R sessions and tools it runs
# from its install directory, which hold files Setup replaces. Unsaved changes
# are lost.
function Stop-RStudio {
  $installDirs = @("${env:ProgramFiles}\RStudio\")
  foreach ($key in @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RStudio',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\RStudio'
  )) {
    $entry = Get-ItemProperty $key -ErrorAction SilentlyContinue
    $location = $entry.InstallLocation
    # Installers that predate NsisMultiUser left InstallLocation unset. Setup
    # finds and upgrades those installs through UninstallString, an unquoted
    # path to Uninstall.exe.
    if (-not $location -and $entry.UninstallString -match '^"?(.+)\\[^\\]+$') {
      $location = $matches[1]
    }
    if ($location) { $installDirs += $location.Trim('"').TrimEnd('\') + '\' }
  }

  $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
    $path = $_.Path
    $_.Name -eq 'rstudio' -or
      ($path -and ($installDirs | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) }))
  })
  foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
  }
  $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
}

try {

Stop-RStudio

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /allusers"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# RStudio can be relaunched between the stop above and Setup's check. Setup
# aborts before changing anything, so stop it again and retry once.
if ($exitCode -eq 2) {
  Write-Host "Install exit code: 2. Retrying in 10 seconds."
  Start-Sleep -Seconds 10
  Stop-RStudio
  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
