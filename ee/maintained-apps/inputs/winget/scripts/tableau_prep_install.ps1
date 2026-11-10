# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Tableau Prep Builder ships as a WiX Burn bundle.
#   ACCEPTEULA=1            - required; the install fails without it
#   SKIPAPPLICATIONLAUNCH=1 - don't open Prep Builder after installing
#
# Each release family (2024.2, 2024.3, ...) installs side by side, so after a
# successful install this removes older Tableau Prep Builder releases, like
# winget's uninstallPrevious. Otherwise the older release keeps the patch policy
# failing.

$exeFilePath = "${env:INSTALLER_PATH}"
# 3010 and 1641 mean success with a reboot required to finish.
$successCodes = @(0, 3010, 1641)

$arpPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# [version] sorts 24.3.41107 below 24.3.41107.0, so compare four-part versions.
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

$olderReleases = @(foreach ($arpPath in $arpPaths) {
  Get-ItemProperty "$arpPath\*" -ErrorAction SilentlyContinue | Where-Object {
    $version = ConvertTo-FullVersion $_.DisplayVersion
    $_.DisplayName -like 'Tableau Prep Builder 20*' -and $_.BundleCachePath -and
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
