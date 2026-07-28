# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# GnuPG's installer starts its background daemons (gpg-agent, dirmngr, keyboxd,
# scdaemon) as part of the install and leaves them resident. PowerShell's
# "Start-Process -Wait" waits for the process *and all of its descendants*, so
# those daemons keep the install script blocked indefinitely. Start without
# -Wait, wait on the installer process itself with a timeout, then stop the
# daemons so the script can return -- the same approach ollama_install.ps1 uses.
$daemons = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf")
$installTimeoutSeconds = 300
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-GnuPGRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "GNU Privacy Guard*" } |
        Select-Object -First 1)
}

try {

# GnuPG uses an NSIS installer.
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# The installer can return before the Add/Remove Programs entry is written; wait
# for it so software inventory sees a complete install.
$elapsed = 0
while (-not (Test-GnuPGRegistered) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for GnuPG to register... ($elapsed seconds)"
}

# Stop the resident daemons the installer started. Leaving them running holds
# file locks that make a later uninstall fail.
foreach ($daemon in $daemons) {
  Stop-Process -Name $daemon -Force -ErrorAction SilentlyContinue
}

if (-not (Test-GnuPGRegistered)) {
  Write-Host "GnuPG did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
