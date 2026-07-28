# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The Gpg4win installer process does not exit on its own on a headless machine,
# and killing it is not enough: like GnuPG's installer (which it bundles), its
# NSIS script writes the Add/Remove Programs entry in a late section, so an
# installer stopped part-way leaves files on disk with no registry entry and
# nothing for inventory to match.
#
# What stalls it is a modal dialog. The NSIS script has MessageBox calls with no
# /SD default -- most relevantly the GpgEX shell-extension registration failure
# ("regsvr32 /s gpgex.dll"), which is exactly the kind of thing that fails with no
# interactive desktop. With nobody to click OK, the installer waits forever.
#
# So: leave the installer's children alone (killing regsvr32 would *guarantee*
# that dialog), and instead close any window the installer puts up. Closing an
# MB_OK dialog is equivalent to acknowledging it, and the install then runs on to
# the section that writes the registry entry. Each poll also logs what is on
# screen and which children are alive, so a future failure is diagnosable from
# the CI log alone.
$leftovers = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "kleopatra", "gpgme-w32spawn")
$installTimeoutSeconds = 420
$pollSeconds = 10
# Let the installer get on with it before we start closing windows, so a dialog
# that is genuinely transient isn't dismissed prematurely.
$graceSeconds = 30

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# The uninstall info is written with SHCTX, so check the per-user hive too in case
# the installer resolves to current-user mode.
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-Gpg4winRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
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

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  $process.Refresh()
  if ($process.HasExited) { break }

  $children = @(Get-Process -Name $leftovers -ErrorAction SilentlyContinue |
    Select-Object -ExpandProperty Name -Unique)
  $windowTitle = ""
  try { $windowTitle = $process.MainWindowTitle } catch { }

  Write-Host "Installing... ($elapsed seconds, registered: $(Test-Gpg4winRegistered), window: '$windowTitle', children: $($children -join ', '))"

  if ($elapsed -ge $graceSeconds -and $process.MainWindowHandle -ne [IntPtr]::Zero) {
    Write-Host "Installer is showing a window ('$windowTitle'); closing it so the install can continue."
    $null = $process.CloseMainWindow()
  }
}

if (-not $process.HasExited) {
  Write-Host "Installer still running after ${installTimeoutSeconds}s; stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
} else {
  Write-Host "Install exit code: $($process.ExitCode)"
}

# Stop the resident processes the installer started. Leaving them running holds
# file locks that make a later uninstall fail.
foreach ($name in $leftovers) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

# Registration is the success signal, not the exit code: on the timeout path the
# installer was killed, so its exit code says nothing about the install.
if (-not (Test-Gpg4winRegistered)) {
  Write-Host "Gpg4win did not register in Add/Remove Programs."
  Exit 1
}

Write-Host "Gpg4win is registered in Add/Remove Programs."
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
