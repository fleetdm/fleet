# Attempts to locate Amazon Chime's uninstaller from the registry and execute it silently.
# Amazon Chime installs per-user, so its uninstall entry lives in HKCU.

$displayName = "Amazon Chime"
$publisher = "Amazon.com Services LLC"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall) {
  Write-Host "Uninstall entry not found for $displayName"
  Exit 0
}

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

# Stop Chime if running so the uninstaller can complete.
Stop-Process -Name "Amazon Chime" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "chime" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Parse the uninstall string defensively into an executable + existing args.
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

# Ensure a silent switch is present.
$uninstallArgs = $existingArgs
if ($uninstallArgs -notmatch '(?i)--silent' -and $uninstallArgs -notmatch '(?i)/S\b') {
    $uninstallArgs = "$uninstallArgs --silent".Trim()
}

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $exe
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = $uninstallArgs
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
