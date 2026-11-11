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
} catch {
  Write-Host "Error listing running processes: $_"
  Exit 1
}
foreach ($proc in $running) {
  Write-Host "Stopping $($proc.Name) (PID $($proc.ProcessId)) before uninstalling."
  Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
}
if ($running) {
  Wait-Process -Id $running.ProcessId -Timeout 30 -ErrorAction SilentlyContinue
  $survivors = @(Get-Process -Id $running.ProcessId -ErrorAction SilentlyContinue)
  if ($survivors) {
    Write-Host "Couldn't stop $($survivors.Name -join ', ') (PID $($survivors.Id -join ', ')), so not uninstalling."
    Exit 1
  }
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
