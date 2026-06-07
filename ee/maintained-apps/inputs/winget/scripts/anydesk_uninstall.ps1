# Attempts to locate AnyDesk's uninstaller from the registry and execute it silently.
# AnyDesk's uninstaller (AnyDesk.exe) supports the documented switches
# "--remove --silent" for an unattended uninstall.

$displayName = "AnyDesk"
$publisher = "AnyDesk Software GmbH"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall) {
  Write-Host "Uninstall entry not found for $displayName"
  Exit 0
}

# Prefer QuietUninstallString when present; otherwise fall back to UninstallString.
$uninstallCommand = if ($uninstall.QuietUninstallString) {
  $uninstall.QuietUninstallString
} elseif ($uninstall.UninstallString) {
  $uninstall.UninstallString
} else {
  $null
}

if (-not $uninstallCommand) {
  Write-Host "No usable uninstall string for $displayName"
  Exit 0
}

# Stop AnyDesk if running so the uninstaller can remove its files/service.
Stop-Process -Name "AnyDesk" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Parse the uninstall string defensively into an executable + existing args.
# Registry uninstall strings come in three shapes: quoted, unquoted-with-spaces,
# and bare tokens (e.g. MsiExec.exe /X{GUID}).
$exe = $null
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exe = $matches[1]
    $existingArgs = $matches[2].Trim()
}

if (-not $exe) {
    Write-Host "Error: Could not parse uninstall command: $uninstallCommand"
    Exit 1
}

# Ensure the silent uninstall switches are present (AnyDesk: --remove --silent).
$uninstallArgs = $existingArgs
if ($uninstallArgs -notmatch '(?i)--remove') {
    $uninstallArgs = "$uninstallArgs --remove".Trim()
}
if ($uninstallArgs -notmatch '(?i)--silent') {
    $uninstallArgs = "$uninstallArgs --silent".Trim()
}

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $exe
        ArgumentList = $uninstallArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
