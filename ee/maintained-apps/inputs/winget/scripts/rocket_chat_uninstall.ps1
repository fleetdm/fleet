# Locates Rocket.Chat from the registry and runs its uninstaller silently.
#
# Rocket.Chat is an electron-builder NSIS app: its machine-wide ARP entry
# registers DisplayName "Rocket.Chat <version>" and Publisher "Rocket.Chat
# Support". The UninstallString points at the NSIS uninstaller; /S runs it
# silently and /allusers removes the per-machine (HKLM) registration.
# Per-user copies are not removed: their uninstallers sit in user-writable
# folders and are never run as SYSTEM. The script reports them and exits 1.

$displayNameLike = "Rocket.Chat*"
$publisherLike = "Rocket.Chat Support*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-PerUserExitCode {
  $entries = @()
  foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.PSChildName -notmatch '^(S-1-5-18|S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)$') { continue }
    foreach ($view in @('SOFTWARE\Microsoft', 'SOFTWARE\WOW6432Node\Microsoft')) {
      $entries += @(Get-ItemProperty "Registry::$($hive.Name)\$view\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
      })
    }
  }
  if ($entries.Count -eq 0) { return 0 }
  Write-Host "Rocket.Chat is still installed per user. Uninstall it as that user:"
  foreach ($e in $entries) { Write-Host "  - $($e.DisplayName) [$($e.PSPath -replace '^.*::', '')]" }
  return 1
}

try {

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit (Get-PerUserExitCode)
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} else {
    $uninstall.UninstallString
}

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

if ($existingArgs -notmatch '(?i)(^|\s)/S($|\s)') {
    $existingArgs = ("$existingArgs /S").Trim()
}
if ($existingArgs -notmatch '(?i)/allusers') {
    $existingArgs = ("$existingArgs /allusers").Trim()
}

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    ArgumentList = $existingArgs
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"
$perUserExitCode = Get-PerUserExitCode
if ($exitCode -eq 0) { $exitCode = $perUserExitCode }
Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
