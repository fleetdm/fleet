# Locates Vernier Spectral Analysis's NSIS uninstaller from the registry and runs it silently.

$displayName = "Vernier Spectral Analysis"
$publisherLike = "Vernier"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisherLike -eq "" -or $_.Publisher -like "*$publisherLike*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallCommand = $uninstall.UninstallString

$exePath = ""
if ($uninstallCommand -match '^\s*"([^"]+)"') {
  $exePath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
  $exePath = $matches[1]
} else {
  $exePath = $uninstallCommand.Trim()
}

$installDir = ""
if ($uninstall.InstallLocation -and (Test-Path $uninstall.InstallLocation)) {
  $installDir = $uninstall.InstallLocation.TrimEnd('\')
} else {
  $installDir = Split-Path -Parent $exePath
}

Write-Host "Uninstall command: $exePath"

try {
  $processOptions = @{
    FilePath = $exePath
    ArgumentList = @("/S", "_?=$installDir")
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"

  Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue

  Exit $exitCode
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
