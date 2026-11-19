# Uninstall Vivaldi (machine-wide Chromium-based browser).
# Looks up the uninstall entry under HKLM (machine install) with an HKCU
# fallback, then runs the Chromium uninstaller with --force-uninstall.

$displayName = "Vivaldi"
$publisher = "Vivaldi Technologies AS."

# Install is machine-wide (--system-level), which registers under HKLM, so look
# there first. HKCU is only a fallback for a stale user-level install.
$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found for '$displayName'"
  Exit 0
}

function Get-ProductName($path) {
  try { [Diagnostics.FileVersionInfo]::GetVersionInfo($path).ProductName } catch { $null }
}

# Stop Vivaldi and the helpers it runs from its install directory. Also stop any
# Vivaldi installer left running by an install that never finished: while it
# runs, the uninstaller fails with exit code 60.
$installDirs = @("${env:ProgramFiles}\Vivaldi\Application\")
if ($uninstall.InstallLocation) {
  $installDirs += $uninstall.InstallLocation.Trim('"').TrimEnd('\') + '\'
}
try {
  $running = @(Get-CimInstance Win32_Process -ErrorAction Stop | Where-Object {
    $path = $_.ExecutablePath
    $path -and (
      ($installDirs | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) }) -or
      ($_.CommandLine -match '\s--system-level\b' -and (Get-ProductName $path) -eq 'Vivaldi Installer')
    )
  })
  if ($running) {
    Write-Host "Stopping $(($running.Name | Sort-Object -Unique) -join ', ') ($($running.Count) processes) before uninstalling."
    $stopped = @{}
    foreach ($proc in $running) {
      $stopped["$($proc.ProcessId)|$($proc.CreationDate.Ticks)"] = $true
      Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
    }
    # Wait-Process can return while stopped processes are still exiting, so poll.
    # Match on start time too, since Windows reuses PIDs.
    $sw = [Diagnostics.Stopwatch]::StartNew()
    do {
      Start-Sleep -Milliseconds 500
      $survivors = @(Get-CimInstance Win32_Process -ErrorAction Stop |
        Where-Object { $stopped["$($_.ProcessId)|$($_.CreationDate.Ticks)"] })
    } while ($survivors -and $sw.Elapsed.TotalSeconds -lt 30)
    if ($survivors) {
      Write-Host "Couldn't stop $(($survivors.Name | Sort-Object -Unique) -join ', ') (PID $($survivors.ProcessId -join ', ')), so not uninstalling."
      Exit 1
    }
  }
} catch {
  Write-Host "Error stopping Vivaldi: $_"
  Exit 1
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
  $uninstall.QuietUninstallString
} else {
  $uninstall.UninstallString
}

# Parse the executable + trailing args, handling the three registry shapes:
# quoted, unquoted-with-spaces (capture through .exe), and a bare token.
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
  $exe = $Matches[1]
  $existingArgs = $Matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
  $exe = $Matches[1]
  $existingArgs = $Matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
  $exe = $Matches[1]
  $existingArgs = $Matches[2].Trim()
} else {
  Write-Host "Unable to parse uninstall command: $uninstallCommand"
  Exit 1
}

# Chromium-based uninstaller flags
$uninstallArgs = "$existingArgs --uninstall --force-uninstall".Trim()

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $uninstallArgs"

try {
  $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -NoNewWindow -PassThru -Wait
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"

  # Chromium uninstallers return 19 on success
  if ($exitCode -eq 0 -or $exitCode -eq 19) {
    Exit 0
  }
  Exit $exitCode
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
