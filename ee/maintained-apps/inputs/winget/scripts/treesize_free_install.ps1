# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# TreeSize Free uses Inno Setup. The switches below are the machine-scope
# "Silent" switches from the winget manifest; /SP- additionally suppresses the
# "This will install..." prompt.
#
# On a headless machine the installer process sits doing nothing and never
# registers in Add/Remove Programs, which is the signature of a modal dialog
# waiting for input that nobody can give. So rather than only waiting it out:
#   * close any window the installer puts up, which acknowledges an OK-style
#     dialog and lets the install continue;
#   * pass Inno's /LOG so the install's own log can be printed if it still fails,
#     which says exactly which step stalled;
#   * log the window title and live child processes on every poll.
# Wait on the installer process alone, too: "Start-Process -Wait" waits for the
# process *and all of its descendants*, so an Inno post-install "run" task
# launching TreeSize would block regardless.
$installTimeoutSeconds = 420
$pollSeconds = 15
$graceSeconds = 30
$leftovers = @("TreeSize", "TreeSizeFree")

$innoLog = Join-Path $env:TEMP "treesize-free-install.log"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-TreeSizeRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "TreeSize Free*" } |
        Select-Object -First 1)
}

function Write-InnoLog {
    if (Test-Path $innoLog) {
        Write-Host "--- Inno Setup log (last 60 lines) ---"
        Get-Content $innoLog -Tail 60 | ForEach-Object { Write-Host $_ }
        Write-Host "--- end of Inno Setup log ---"
    } else {
        Write-Host "No Inno Setup log was written at $innoLog."
    }
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /AllUsers /LOG=`"$innoLog`"" `
  -PassThru
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

  Write-Host "Installing... ($elapsed seconds, registered: $(Test-TreeSizeRegistered), window: '$windowTitle', children: $($children -join ', '))"

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

foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

# Registration is the success signal, not the exit code: on the timeout path the
# installer was killed, so its exit code says nothing about the install.
if (-not (Test-TreeSizeRegistered)) {
  Write-Host "TreeSize Free did not register in Add/Remove Programs."
  Write-InnoLog
  Exit 1
}

Write-Host "TreeSize Free is registered in Add/Remove Programs."
Exit 0

} catch {
  Write-Host "Error: $_"
  Write-InnoLog
  Exit 1
}
