$displayName = "Enpass"
$publisher   = "Enpass Technologies Inc."
$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}
if (-not $uninstall) { Write-Host "Uninstall entry not found"; Exit 0 }
# Prefer the WiX burn QuietUninstallString when present
$uninstallString = if ($uninstall.QuietUninstallString) { $uninstall.QuietUninstallString } else { $uninstall.UninstallString }
if (-not $uninstallString) { Write-Host "Uninstall string not found"; Exit 0 }
$exePath = ""
$extraArgs = ""
if ($uninstallString -match '^"([^"]+)"(.*)') { $exePath = $matches[1]; $extraArgs = $matches[2].Trim() }
elseif ($uninstallString -match '^(.+?\.exe)(.*)$') { $exePath = $matches[1]; $extraArgs = $matches[2].Trim() }
else { Write-Host "Error: Could not parse uninstall string: $uninstallString"; Exit 1 }
$argumentList = @()
if ($extraArgs) { $argumentList += $extraArgs }
if ($argumentList -notmatch "uninstall") { $argumentList += "/uninstall" }
$argumentList += @("/quiet", "/norestart")
try {
  $processOptions = @{ FilePath = $exePath; ArgumentList = $argumentList; NoNewWindow = $true; PassThru = $true; Wait = $true }
  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
  Exit $exitCode
} catch { Write-Host "Error running uninstaller: $_"; Exit 1 }
