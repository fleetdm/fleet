# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Dell Display and Peripheral Manager ships an InstallShield setup (~425 MB) that
# extracts to %TEMP%, hands off to child processes, and leaves its own background
# app running once installed. PowerShell's "Start-Process -Wait" waits for the
# process *and all of its descendants*, so those would block this script. Wait on
# the installer process alone, poll for the Add/Remove Programs entry (the
# hand-off means the parent can exit before the install is finished), then stop
# what the installer left running.
# Switch choice: the winget manifest lists "/Silent", but that produced exit code
# -2147213312 (0x80042000) with nothing installed. This is an InstallShield setup,
# whose actual silent switch is "/S" -- which is also what ManageEngine's silent
# install reference documents for DDPM 2.2.2.8. "/CreateDebugLog" is Dell's own
# logging switch, so a future failure is diagnosable from the CI log.
$installTimeoutSeconds = 480
$pollSeconds = 15
$leftovers = @("DDPM", "DellDisplayManager", "DisplayManager")

$dellLog = Join-Path $env:TEMP "ddpm-install.log"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Write-DellLog {
    if (Test-Path $dellLog) {
        Write-Host "--- DDPM install log (last 60 lines) ---"
        Get-Content $dellLog -Tail 60 | ForEach-Object { Write-Host $_ }
        Write-Host "--- end of DDPM install log ---"
    } else {
        Write-Host "No DDPM install log was written at $dellLog."
    }
}

function Test-DdpmRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Dell Display and Peripheral Manager*" } |
        Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/S /CreateDebugLog=`"$dellLog`"" `
  -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Installing... ($elapsed seconds, registered: $(Test-DdpmRegistered))"
}

if (-not $process.HasExited) {
  $registered = Test-DdpmRegistered
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
  foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

  if ($registered) {
    Write-Host "Installer still running after ${installTimeoutSeconds}s but DDPM is registered; stopped the lingering process."
    Exit 0
  }

  Write-Host "Installer did not finish within ${installTimeoutSeconds}s and DDPM is not registered."
  Write-DellLog
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# InstallShield's parent process can exit before the child finishes writing the
# Add/Remove Programs entry, so give it a bounded window to appear.
$elapsed = 0
while (-not (Test-DdpmRegistered) -and ($elapsed -lt 180)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Waiting for DDPM to register... ($elapsed seconds)"
}

foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

if (-not (Test-DdpmRegistered)) {
  Write-Host "Dell Display and Peripheral Manager did not register in Add/Remove Programs."
  Write-DellLog
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
