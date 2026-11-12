# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The installer stalls on a modal dialog with no interactive desktop and never
# exits. Closing its window lets it run through to the section that writes the
# Add/Remove Programs entry; killing it instead would leave a partial install.
$daemons = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "gpa", "launch-gpa")
$installTimeoutSeconds = 420
$pollSeconds = 10
$graceSeconds = 30

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# Uninstall info is written with SHCTX, so it can land per-user.
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-GnuPGRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "GNU Privacy Guard*" } |
        Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
# Keeps .ExitCode readable after the process ends.
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

# Stop the resident daemons; they hold file locks the uninstall needs released.
foreach ($name in $daemons) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

# Registration is the success signal: a killed installer's exit code says nothing.
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
