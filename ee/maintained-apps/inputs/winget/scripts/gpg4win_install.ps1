# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Gpg4win bundles GnuPG, so the install starts the same resident daemons
# (gpg-agent, dirmngr, keyboxd, scdaemon) and can leave Kleopatra running.
# PowerShell's "Start-Process -Wait" waits for the process *and all of its
# descendants*, so those keep the install script blocked indefinitely. Start
# without -Wait, wait on the installer process itself with a timeout, then stop
# the leftovers -- the same approach ollama_install.ps1 uses.
$leftovers = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "kleopatra", "gpgme-w32spawn")
$installTimeoutSeconds = 300
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-Gpg4winRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Gpg4win*" } |
        Select-Object -First 1)
}

try {

# Gpg4win uses an NSIS installer.
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
while (-not (Test-Gpg4winRegistered) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for Gpg4win to register... ($elapsed seconds)"
}

# Stop the resident processes the installer started. Leaving them running holds
# file locks that make a later uninstall fail.
foreach ($name in $leftovers) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

if (-not (Test-Gpg4winRegistered)) {
  Write-Host "Gpg4win did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
