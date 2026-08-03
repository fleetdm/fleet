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
  # Stop the bootstrapper only. A child msiexec may still be mid-transaction, and
  # killing that would leave a half-installed product; letting it finish gives the
  # registration poll below a chance to confirm the install either way.
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

# A killed process's code says nothing about the install, so leave it unset.
$exitCode = $null
if (-not $killed -and $process.HasExited) {
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
  # Surface the installer's own failure code when it reported one.
  if ($null -ne $exitCode -and $exitCode -ne 0) { Exit $exitCode }
  Exit 1
}
Write-Host "Registered '$($entry.DisplayName)' by '$($entry.Publisher)', version $($entry.DisplayVersion)."

# Registration is the success signal, and it is the same entry Fleet's detection
# query reads. A non-zero code after a confirmed registration (3010 and 1641 are
# reboot-required successes; 1638 means the engine is already present) is logged
# rather than reported as a failed install.
if ($null -ne $exitCode -and @(0, 3010, 1641) -notcontains $exitCode) {
  Write-Host "Installer returned $exitCode but the product is registered; treating the install as successful."
}

Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
