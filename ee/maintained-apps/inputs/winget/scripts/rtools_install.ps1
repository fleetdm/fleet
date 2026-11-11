# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Rtools extracts ~52,000 files to C:\rtools45, which can take well over 10
# minutes with real-time antivirus scanning. The limit stays under Fleet's
# 1-hour script timeout so a stuck install still reports why it failed.
$installTimeoutSeconds = 3300
$pollSeconds = 15

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-RtoolsEntries {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Rtools*" } |
        ForEach-Object { "$($_.PSChildName)|$($_.DisplayVersion)" }
}

function Test-RtoolsRegistered {
    $null -ne (Get-RtoolsEntries | Select-Object -First 1)
}

# An Rtools entry left by an earlier install doesn't mean this one finished.
$preexistingEntries = @(Get-RtoolsEntries)

function Test-RtoolsRegisteredByThisInstall {
    $null -ne (Get-RtoolsEntries | Where-Object { $preexistingEntries -notcontains $_ } | Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
  -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

# Wall-clock time, since polling falls behind on a busy host.
$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
$nextLogSeconds = 60
while (-not $process.HasExited -and ($stopwatch.Elapsed.TotalSeconds -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed = [int]$stopwatch.Elapsed.TotalSeconds
  if ($elapsed -ge $nextLogSeconds) {
    Write-Host "Installing... ($elapsed seconds, registered: $(Test-RtoolsRegisteredByThisInstall))"
    $nextLogSeconds = $elapsed + 60
  }
}
$elapsed = [int]$stopwatch.Elapsed.TotalSeconds

if (-not $process.HasExited) {
  # Registered means the files are in place and only a lingering process remains.
  if (Test-RtoolsRegisteredByThisInstall) {
    Write-Host "Installer still running after ${installTimeoutSeconds}s but Rtools is registered; stopping the lingering process."
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    Exit 0
  }

  Write-Host "Installer did not finish within ${installTimeoutSeconds}s and Rtools is not registered."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode after $elapsed seconds"

# The parent can exit while a descendant is still writing the ARP entry.
$settle = 0
while (-not (Test-RtoolsRegistered) -and ($settle -lt 90)) {
  Start-Sleep -Seconds $pollSeconds
  $settle += $pollSeconds
  Write-Host "Waiting for Rtools to register... ($settle seconds)"
}

if (-not (Test-RtoolsRegistered)) {
  Write-Host "Rtools did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
