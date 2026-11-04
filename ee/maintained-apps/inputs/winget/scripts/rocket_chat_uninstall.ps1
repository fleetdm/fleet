# Locates Rocket.Chat from the registry and runs its uninstaller silently.
#
# Rocket.Chat is an electron-builder NSIS app: its machine-wide ARP entry
# registers DisplayName "Rocket.Chat <version>" and Publisher "Rocket.Chat
# Support". The UninstallString points at the NSIS uninstaller; /S runs it
# silently and /allusers removes the per-machine (HKLM) registration.

$displayNameLike = "Rocket.Chat*"
$publisherLike = "Rocket.Chat Support*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

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
  Exit 0
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
Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
