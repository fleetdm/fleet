# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# TreeSize Free uses Inno Setup. The switches below are the machine-scope
# "Silent" switches from the winget manifest, including /AllUsers. What made the
# previous script hang was PowerShell rather than the switches: "Start-Process
# -Wait" waits for the process *and all of its descendants*, so anything the
# installer leaves running (an Inno post-install "run" task launching TreeSize,
# for example) blocks the script indefinitely. Wait on the installer process
# alone, log progress, then stop any app the installer started.
$installTimeoutSeconds = 420
$pollSeconds = 15
$leftovers = @("TreeSize", "TreeSizeFree")

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-TreeSizeRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "TreeSize Free*" } |
        Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /AllUsers" `
  -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Installing... ($elapsed seconds, registered: $(Test-TreeSizeRegistered))"
}

if (-not $process.HasExited) {
  # Still running at the cap. If TreeSize Free already registered in Add/Remove
  # Programs the install finished and only a lingering child remains, so stopping
  # it is safe; otherwise the install genuinely did not complete.
  $registered = Test-TreeSizeRegistered
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
  foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

  if ($registered) {
    Write-Host "Installer still running after ${installTimeoutSeconds}s but TreeSize Free is registered; stopped the lingering process."
    Exit 0
  }

  Write-Host "Installer did not finish within ${installTimeoutSeconds}s and TreeSize Free is not registered."
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

if (-not (Test-TreeSizeRegistered)) {
  Write-Host "TreeSize Free did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
