# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# NVDA writes DisplayName as "NVDA <version>".
function Test-NvdaRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object {
            $_.DisplayName -and
            ($_.DisplayName -eq 'NVDA' -or $_.DisplayName -like 'NVDA *') -and
            $_.Publisher -like '*NV Access*'
        } |
        Select-Object -First 1)
}

try {

if (-not (Test-Path $exeFilePath)) {
    Write-Host "Error: Installer file not found at: $exeFilePath"
    Exit 1
}

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "--install-silent" `
  -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

# NVDA shows a modal "File in Use" box on failure even when silent, which would
# hang forever as SYSTEM.
$killed = $false
if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer did not exit within ${installTimeoutSeconds}s, stopping it."
  Write-Host "NVDA is likely running in another session and the installer is blocked on a dialog."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  # Only the copies the launcher runs from %TEMP%. An installed NVDA runs as
  # nvda.exe; killing that would cut off a signed-in user's screen reader.
  foreach ($name in @('nvda_noUIAccess', 'nvda_uiAccess')) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
  }
  # Reading .ExitCode while the process is alive would throw.
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

$exitCode = $null
if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"
} else {
  Write-Host "Installer could not be stopped; falling back to the registration check."
}

$elapsed = 0
while (-not (Test-NvdaRegistered) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for NVDA to register... ($elapsed seconds)"
}

# NVDA exits 0 even when the install failed, so registration is the real signal.
if (-not (Test-NvdaRegistered)) {
  Write-Host "NVDA did not register in Add/Remove Programs."
  Write-Host "If NVDA was already running for a signed-in user, exit it and retry."
  Exit 1
}

if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
