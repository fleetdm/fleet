# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# The download is a self-extracting package whose setup.cmd forwards its first
# argument to "msiexec /i AceRedist.msi", so /quiet reaches the MSI.

$exeFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-AccessDatabaseEngineEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq "Microsoft Access database engine 2016 (English)" } |
        Select-Object -First 1
}

try {

# The 64-bit redistributable refuses to install alongside 32-bit Office. Check
# first so this fails with an actionable message rather than an installer error.
$officeConfig = 'HKLM:\SOFTWARE\Microsoft\Office\ClickToRun\Configuration'
$officePlatform = (Get-ItemProperty -Path $officeConfig -Name Platform -ErrorAction SilentlyContinue).Platform
if ($officePlatform -eq 'x86') {
  Write-Host "32-bit Microsoft Office is installed on this host (Click-to-Run platform: x86)."
  Write-Host "The 64-bit Access Database Engine cannot be installed alongside it. Use the 32-bit redistributable instead."
  Exit 1
}

# -Wait also waits on descendants, so wait on the installer process alone.
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/quiet" -PassThru
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
while (-not (Get-AccessDatabaseEngineEntry) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for the Access Database Engine to register... ($elapsed seconds)"
}

$entry = Get-AccessDatabaseEngineEntry
if (-not $entry) {
  Write-Host "The Access Database Engine did not register in Add/Remove Programs."
  Exit 1
}
Write-Host "Registered '$($entry.DisplayName)' by '$($entry.Publisher)', version $($entry.DisplayVersion)."

# Registration above is the success signal; a killed process's code means nothing.
if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
