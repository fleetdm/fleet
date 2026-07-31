# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-HandBrakeEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "HandBrake *" } |
        Select-Object -First 1
}

try {

# HandBrake will not run without the .NET Desktop Runtime, and the installer does
# not bundle or install it. Fail here with an actionable message rather than
# leaving behind an app that installs but cannot start.
$desktopRuntimeRoot = Join-Path $env:ProgramFiles "dotnet\shared\Microsoft.WindowsDesktop.App"
$hasDesktopRuntime = (Test-Path $desktopRuntimeRoot) -and
    (Get-ChildItem -Path $desktopRuntimeRoot -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like "10.*" } | Select-Object -First 1)

if (-not $hasDesktopRuntime) {
  Write-Host "HandBrake requires the Microsoft .NET Desktop Runtime 10, which was not found at $desktopRuntimeRoot."
  Write-Host "Install the .NET Desktop Runtime 10 on this host, then retry."
  Exit 1
}

# -Wait also waits on descendants, so wait on the installer process alone.
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

$killed = $false
if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

$exitCode = $null
if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"
}

# The installer can return before the ARP entry is written.
$elapsed = 0
while (-not (Get-HandBrakeEntry) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for HandBrake to register... ($elapsed seconds)"
}

$entry = Get-HandBrakeEntry
if (-not $entry) {
  Write-Host "HandBrake did not register in Add/Remove Programs."
  Exit 1
}
Write-Host "Registered '$($entry.DisplayName)', version $($entry.DisplayVersion)."

# Registration above is the success signal; a killed process's code means nothing.
if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
