# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The installer stalls on a modal dialog with no interactive desktop and never
# exits. Closing the window lets it run through to the section that writes the
# Add/Remove Programs entry; killing it instead would leave a partial install.
# The dialog belongs to a child process, so search the whole tree.
$leftovers = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "kleopatra", "gpgme-w32spawn")
$installTimeoutSeconds = 420
$pollSeconds = 10
$graceSeconds = 30

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# Uninstall info is written with SHCTX, so it can land per-user.
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

# The installer process plus its descendants.
function Get-InstallerTree([int]$rootId) {
    $all = @{}
    Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        ForEach-Object { $all[[int]$_.ProcessId] = [int]$_.ParentProcessId }

    $ids = New-Object System.Collections.Generic.HashSet[int]
    $null = $ids.Add($rootId)
    for ($depth = 0; $depth -lt 5; $depth++) {
        foreach ($procId in @($all.Keys)) {
            if ($ids.Contains($all[$procId])) { $null = $ids.Add($procId) }
        }
    }

    Get-Process -ErrorAction SilentlyContinue | Where-Object { $ids.Contains($_.Id) }
}

function Test-Gpg4winRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Gpg4win*" } |
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

  $tree = Get-InstallerTree $process.Id
  $names = @($tree | Select-Object -ExpandProperty ProcessName -Unique)
  $windowTitle = ""
  try { $windowTitle = $process.MainWindowTitle } catch { }
  $childWindows = @($tree | Where-Object { $_.MainWindowHandle -ne [IntPtr]::Zero } |
    ForEach-Object { "$($_.ProcessName): '$($_.MainWindowTitle)'" })

  Write-Host "Installing... ($elapsed seconds, registered: $(Test-Gpg4winRegistered), window: '$windowTitle', tree: $($names -join ', '), child windows: $($childWindows -join ' | '))"

  if ($elapsed -ge $graceSeconds) {
    foreach ($p in $tree) {
      if ($p.MainWindowHandle -ne [IntPtr]::Zero) {
        Write-Host "Closing window owned by $($p.ProcessName) ('$($p.MainWindowTitle)') so the install can continue."
        $null = $p.CloseMainWindow()
      }
    }
  }
}

if (-not $process.HasExited) {
  Write-Host "Installer still running after ${installTimeoutSeconds}s; stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
} else {
  Write-Host "Install exit code: $($process.ExitCode)"
}

# Stop resident processes; they hold file locks the uninstall needs released.
foreach ($name in $leftovers) {
  Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

# Registration is the success signal: a killed installer's exit code says nothing.
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
