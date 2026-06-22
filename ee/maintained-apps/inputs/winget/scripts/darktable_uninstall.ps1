# darktable ships as an Inno Setup installer (NOT NSIS, despite winget's
# metadata), so its uninstaller (unins000.exe) needs Inno's silent switches.
# The old NSIS installer registered DisplayName "darktable"; the current Inno
# installer registers a versioned DisplayName (e.g. "darktable 5.6.0") with
# Publisher "darktable team" (older metadata used "the darktable project").
# Match both, scoped to a darktable publisher.

$displayName = "darktable"
# Substring match on publisher covers "darktable team" and "the darktable project".
$publisher   = "darktable"
$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName *") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}
if (-not $uninstall -or -not $uninstall.UninstallString) { Write-Host "Uninstall entry not found"; Exit 0 }
$uninstallString = $uninstall.UninstallString
$exePath = ""
if ($uninstallString -match '^"([^"]+)"(.*)') { $exePath = $matches[1] }
elseif ($uninstallString -match '^(.+?\.exe)(.*)$') { $exePath = $matches[1] }
else { Write-Host "Error: Could not parse uninstall string: $uninstallString"; Exit 1 }
$installDir = if ($uninstall.InstallLocation -and (Test-Path -LiteralPath $uninstall.InstallLocation)) { $uninstall.InstallLocation.TrimEnd('\') } else { (Split-Path -Parent $exePath).TrimEnd('\') }
# Inno Setup uninstaller silent switches. The NSIS-style "/S _?=<dir>" does not
# apply: Inno's uninstaller runs in place, so Start-Process -Wait waits correctly.
$argumentList = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
try {
  $processOptions = @{ FilePath = $exePath; ArgumentList = $argumentList; NoNewWindow = $true; PassThru = $true; Wait = $true }
  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
  # 3010 = success, reboot required; 1641 = success, reboot initiated.
  if ($exitCode -eq 3010 -or $exitCode -eq 1641) { $exitCode = 0 }
  # Only sweep leftovers on a successful uninstall, and never a root/short path
  if ($exitCode -eq 0 -and $installDir) {
      $resolvedDir = $null
      try { $resolvedDir = (Resolve-Path -LiteralPath $installDir -ErrorAction Stop).Path } catch { $resolvedDir = $null }
      if ($resolvedDir -and ($resolvedDir -match '^[A-Za-z]:\\') -and ((($resolvedDir.TrimEnd('\')) -split '\\').Count -ge 3) -and (Test-Path -LiteralPath $resolvedDir)) {
          Remove-Item -LiteralPath $resolvedDir -Recurse -Force -ErrorAction SilentlyContinue
      }
  }
  Exit $exitCode
} catch { Write-Host "Error running uninstaller: $_"; Exit 1 }
