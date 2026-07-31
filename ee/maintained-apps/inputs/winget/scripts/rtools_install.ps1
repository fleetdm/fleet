# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Rtools unpacks a large toolchain (~460 MB) to C:\rtools45, not Program Files.
# -Wait waits on descendants, so wait on the installer process alone and log
# progress to tell a slow unpack apart from a stuck one.
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
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Installing... ($elapsed seconds, registered: $(Test-RtoolsRegistered))"
}

if (-not $process.HasExited) {
  # Registered means the install finished and only a lingering child remains.
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
