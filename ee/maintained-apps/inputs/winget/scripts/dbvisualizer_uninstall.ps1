$displayName = "DbVisualizer"
$publisher   = "DbVis"
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
$argumentList = @("-q")
try {
  $processOptions = @{ FilePath = $exePath; ArgumentList = $argumentList; NoNewWindow = $true; PassThru = $true; Wait = $true }
  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
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
