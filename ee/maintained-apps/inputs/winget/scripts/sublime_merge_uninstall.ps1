# Locates Sublime Merge's Inno Setup uninstaller from the registry and runs it silently.

$displayName = "Sublime Merge"
$publisher = "Sublime HQ Pty Ltd"

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

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Stop any running Sublime Merge processes before uninstalling.
Stop-Process -Name "sublime_merge" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Prefer QuietUninstallString when present.
$uninstallCommand = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} else {
    $uninstall.UninstallString
}

$silentFlags = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

# Parse the UninstallString defensively (quoted / unquoted-with-spaces / bare token).
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
} else {
    Write-Host "Error: could not parse uninstall command: $uninstallCommand"
    Exit 1
}

if ($existingArgs -match '(?i)/VERYSILENT') {
    $uninstallArgs = $existingArgs
} else {
    $uninstallArgs = ("$existingArgs $silentFlags").Trim()
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
