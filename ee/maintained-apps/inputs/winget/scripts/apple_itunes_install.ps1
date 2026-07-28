# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# iTunes ships as a WiX bundle that chains four MSIs (Apple Application Support
# x86 and x64, Apple Mobile Device Support, Bonjour, and iTunes itself). The
# bundle hands off to child msiexec processes, so the parent can return before
# the install has actually finished. Wait for the Add/Remove Programs entry and
# for msiexec to drain before reporting success, otherwise a following uninstall
# races an install that is still running.
# "/quiet /norestart" is the documented Silent switch set from the winget manifest.
$installTimeoutSeconds = 480
$pollSeconds = 10

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-ITunesRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq "iTunes" } |
        Select-Object -First 1)
}

function Wait-ForMsiexec([int]$maxSeconds) {
    $waited = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($waited -lt $maxSeconds)) {
        Start-Sleep -Seconds 5
        $waited += 5
        Write-Host "Waiting for msiexec to drain... ($waited seconds)"
    }
}

try {

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/quiet /norestart" -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Installing... ($elapsed seconds, registered: $(Test-ITunesRegistered))"
}

if (-not $process.HasExited) {
  Write-Host "Installer did not finish within ${installTimeoutSeconds}s."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# The bundle can exit while its chained MSIs are still being applied.
Wait-ForMsiexec 240

$elapsed = 0
while (-not (Test-ITunesRegistered) -and ($elapsed -lt 120)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  Write-Host "Waiting for iTunes to register... ($elapsed seconds)"
}

# iTunes starts its helper as soon as it is installed; leaving it running holds
# file locks that make a later uninstall fail.
foreach ($name in @("iTunes", "iTunesHelper", "AppleMobileDeviceService", "mDNSResponder")) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

if (-not (Test-ITunesRegistered)) {
  Write-Host "iTunes did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
