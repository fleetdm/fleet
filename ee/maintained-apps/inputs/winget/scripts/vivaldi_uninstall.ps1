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

# Kill any running Vivaldi processes before uninstalling
Stop-Process -Name "vivaldi" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

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
