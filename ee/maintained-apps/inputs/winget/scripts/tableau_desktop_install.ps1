# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Tableau Desktop ships as a WiX Burn bundle.
#   /quiet /norestart   - standard Burn silent install
#   ACCEPTEULA=1        - required by Tableau (winget Custom switch)
#   SKIPAPPLICATIONLAUNCH=1 - don't launch after install
#
# Each release family (2024.2, 2024.3, ...) installs side by side, so after a
# successful install this removes older Tableau Desktop releases, like winget's
# uninstallPrevious. Otherwise the older release keeps the patch policy failing.

$exeFilePath = "${env:INSTALLER_PATH}"
# msiexec reboot-required success codes (Burn bundles propagate these).
$successCodes = @(0, 3010, 1641)

$arpPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# [version] sorts 24.3.965 below 24.3.965.0, so compare four-part versions.
function ConvertTo-FullVersion($text) {
  $v = $null
  if (-not [version]::TryParse("$text", [ref]$v)) { return $null }
  [version]::new($v.Major, $v.Minor, [Math]::Max($v.Build, 0), [Math]::Max($v.Revision, 0))
}

try {

$vi = (Get-Item -LiteralPath $exeFilePath).VersionInfo
$targetVersion = [version]::new($vi.ProductMajorPart, $vi.ProductMinorPart, $vi.ProductBuildPart, $vi.ProductPrivatePart)
Write-Host "Installer version: $targetVersion"

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/quiet /norestart ACCEPTEULA=1 SKIPAPPLICATIONLAUNCH=1"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

if ($successCodes -notcontains $exitCode) {
  Exit $exitCode
}

# Desktop registers as "Tableau 2024.3 (...)"; Reader and Prep Builder names
# don't start with "Tableau 20".
$olderReleases = @(foreach ($arpPath in $arpPaths) {
  Get-ItemProperty "$arpPath\*" -ErrorAction SilentlyContinue | Where-Object {
    $version = ConvertTo-FullVersion $_.DisplayVersion
    $_.DisplayName -like 'Tableau 20*' -and $_.BundleCachePath -and
      (Test-Path -LiteralPath $_.BundleCachePath) -and
      $version -and $version -lt $targetVersion
  }
})

$exitCode = 0
foreach ($release in $olderReleases) {
  Write-Host "Removing older release: $($release.DisplayName)"
  try {
    $p = Start-Process -FilePath $release.BundleCachePath `
      -ArgumentList '/uninstall', '/quiet', '/norestart' `
      -PassThru -Wait
    Write-Host "Uninstall exit code: $($p.ExitCode)"
    if ($successCodes -notcontains $p.ExitCode) { $exitCode = $p.ExitCode }
  } catch {
    Write-Host "Failed to remove $($release.DisplayName): $_"
    $exitCode = 1
  }
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
