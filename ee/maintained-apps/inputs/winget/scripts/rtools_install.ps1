# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Rtools uses Inno Setup and unpacks a large toolchain (a ~460 MB installer) to
# C:\rtools45, not Program Files. Two things made the previous script hang:
# PowerShell's "Start-Process -Wait" waits for the process *and all of its
# descendants*, and the unpack itself is slow. Wait on the installer process
# alone, with a cap below the budget the caller allows, and log progress so a
# slow unpack is distinguishable from a stuck one.
$installTimeoutSeconds = 480
$pollSeconds = 15

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-RtoolsRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Rtools*" } |
        Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
  -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Installing... ($elapsed seconds, registered: $(Test-RtoolsRegistered))"
}

if (-not $process.HasExited) {
  # The installer is still running at the cap. If Rtools has already registered
  # in Add/Remove Programs the install itself finished and what is left is a
  # lingering child, so stopping it is safe; otherwise the unpack genuinely did
  # not finish in time and this is a real failure.
  if (Test-RtoolsRegistered) {
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
Write-Host "Install exit code: $exitCode"

# The parent can exit while a descendant is still finishing the unpack and writing
# the ARP entry -- the same hand-off that makes -Wait block. Give registration a
# bounded window to appear rather than checking once.
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
