# Locates DeepL's uninstaller from the registry and runs it silently.
# DeepL installs via Zero Install (0install). Its QuietUninstallString is a
# 0install command, e.g.:

$displayNameLike = "DeepL*"
$publisher = "DeepL SE"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like "$publisher*"
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "DeepL" -Force -ErrorAction SilentlyContinue

# Prefer QuietUninstallString -- it already carries 0install's silent flags
# (--batch --background). Fall back to UninstallString and add those flags
# (NOT --verysilent) to keep it non-interactive.
$useQuiet = [bool]$uninstall.QuietUninstallString
$uninstallCommand = if ($useQuiet) { $uninstall.QuietUninstallString } else { $uninstall.UninstallString }

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallC
