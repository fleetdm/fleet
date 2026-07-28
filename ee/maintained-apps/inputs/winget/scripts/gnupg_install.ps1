# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The GnuPG installer process does not exit on its own on a headless machine, and
# killing it is not enough: its NSIS script writes the Add/Remove Programs entry
# in the very last, hidden section (inst.nsi -> DisplayName "GNU Privacy Guard"
# under ...\Uninstall\GnuPG), so an installer stopped part-way leaves files on
# disk with no registry entry and nothing for inventory to match.
#
# What stalls it is a modal dialog. inst.nsi has several MessageBox calls with no
# /SD default -- most relevantly the GpgEX shell-extension registration failure
# ("regsvr32 /s gpgex6.dll"), which is exactly the kind of thing that fails with
# no interactive desktop. With nobody to click OK, the installer waits forever.
#
# So: leave the installer's children alone (killing regsvr32 would *guarantee*
# that dialog), and instead close any window the installer puts up. Closing an
# MB_OK dialog is equivalent to acknowledging it, and the install then runs on to
# the section that writes the registry entry. Each poll also logs what is on
# screen and which children are alive, so a future failure is diagnosable from
# the CI log alone.
$daemons = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "gpa", "launch-gpa")
$installTimeoutSeconds = 420
$pollSeconds = 10
# Let the installer get on with it before we start closing windows, so a dialog
# that is genuinely transient isn't dismissed prematurely.
$graceSeconds = 30

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# inst.nsi writes the uninstall info with SHCTX, so check the per-user hive too in
# case the installer resolves to current-user mode.
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-GnuPGRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
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

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  $process.Refresh()
  if ($process.HasExited) { break }

  $children = @(Get-Process -Name $daemons -ErrorAction SilentlyContinue |
    Select-Object -ExpandProperty Name -Unique)
  $windowTitle = ""
  try { $windowTitle = $process.MainWindowTitle } catch { }

  Write-Host "Installing... ($elapsed seconds, registered: $(Test-GnuPGRegistered), window: '$windowTitle', children: $($children -join ', '))"

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

# Stop the resident daemons the installer started. Leaving them running holds
# file locks that make a later uninstall fail.
foreach ($name in $daemons) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

# Registration is the success signal, not the exit code: on the timeout path the
# installer was killed, so its exit code says nothing about the install.
if (-not (Test-GnuPGRegistered)) {
  Write-Host "GnuPG did not register in Add/Remove Programs."
  Exit 1
}

Write-Host "GnuPG is registered in Add/Remove Programs."
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
