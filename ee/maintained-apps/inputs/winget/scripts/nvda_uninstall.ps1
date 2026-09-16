$displayName = 'NVDA'
$publisher = 'NV Access'
$uninstallTimeoutSeconds = 300

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-NvdaUninstallEntry {
  foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -and
      ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName *") -and
      $_.Publisher -like "*$publisher*"
    }
    if ($items) { return ($items | Select-Object -First 1) }
  }
  return $null
}

try {

$uninstall = Get-NvdaUninstallEntry
if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# NVDA's UninstallString is an unquoted path containing spaces.
$uninstallString = $uninstall.UninstallString
$exePath = ""
if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') { $exePath = $matches[1] }
elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { $exePath = $matches[1] }
else { Write-Host "Error: Could not parse uninstall string: $uninstallString"; Exit 1 }

if (-not (Test-Path -LiteralPath $exePath)) {
  Write-Host "Error: Uninstaller not found at: $exePath"
  Exit 1
}

$installDir = $null
foreach ($candidate in @($uninstall.InstallDir, $uninstall.InstallLocation)) {
  if ($candidate -and (Test-Path -LiteralPath $candidate)) {
    $installDir = $candidate.TrimEnd('\')
    break
  }
}
if (-not $installDir) { $installDir = (Split-Path -Parent $exePath).TrimEnd('\') }

# "/S" is the documented silent switch; "_?=" must come last so the NSIS
# uninstaller runs in place instead of returning immediately from a temp copy.
$argumentList = @("/S", "_?=$installDir")

$process = Start-Process -FilePath $exePath -ArgumentList $argumentList -NoNewWindow -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

$killed = $false
if (-not $process.WaitForExit($uninstallTimeoutSeconds * 1000)) {
  Write-Host "Uninstaller did not exit within ${uninstallTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

$exitCode = $null
if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
}

if (Get-NvdaUninstallEntry) {
  Write-Host "NVDA is still registered in Add/Remove Programs after uninstall."
  if ($killed -or $null -eq $exitCode -or $exitCode -eq 0) { Exit 1 }
  Exit $exitCode
}

# NVDA removes its directory with /REBOOTOK, so uninstall.exe can be left behind.
# Sweep it, but never a root or short path.
if ($installDir) {
  $resolvedDir = $null
  try { $resolvedDir = (Resolve-Path -LiteralPath $installDir -ErrorAction Stop).Path } catch { $resolvedDir = $null }
  if ($resolvedDir -and ($resolvedDir -match '^[A-Za-z]:\\') -and
      ((($resolvedDir.TrimEnd('\')) -split '\\').Count -ge 3) -and
      (Test-Path -LiteralPath $resolvedDir)) {
    Remove-Item -LiteralPath $resolvedDir -Recurse -Force -ErrorAction SilentlyContinue
  }
}

Exit 0

} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
