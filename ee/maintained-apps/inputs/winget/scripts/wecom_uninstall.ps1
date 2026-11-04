# Locates WeCom from the registry and runs its uninstaller silently.
#
# WeCom is a Tencent NSIS app. Its machine-wide ARP entry is keyed by the product
# name and is localized: "WeCom" on English Windows, "企业微信" on Chinese. The
# UninstallString points at Uninstall.exe in the install directory (WXWork); /S
# runs it silently. The running app is stopped first so the uninstaller doesn't
# block on it.

$displayNames = @('WeCom', '企业微信')

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $displayNames -contains $_.DisplayName
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "WXWork" -Force -ErrorAction SilentlyContinue

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

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$startArgs = @{
    FilePath = $exePath
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
}
if ($existingArgs) { $startArgs.ArgumentList = $existingArgs }

$process = Start-Process @startArgs
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"
Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
