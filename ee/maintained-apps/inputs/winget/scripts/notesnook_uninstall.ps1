# Uninstall script for Notesnook (NSIS / electron-builder)
# Registry DisplayName is versioned (e.g. "Notesnook 3.3.23"), so match with a wildcard.
$displayNameLike = "Notesnook*"
$publisher = "Streetwriters"

$hkcuKeys = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKCU:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)
$hklmKeys = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$exitCode = 0

try {

$key = $null
foreach ($scope in @($hkcuKeys, $hklmKeys)) {
  $key = Get-ChildItem -Path $scope -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath } |
    Where-Object {
      ($_.DisplayName -like $displayNameLike) -and
      ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
    } | Select-Object -First 1
  if ($key) { break }
}

if (-not $key) {
  Write-Host "Uninstall entry not found for '$displayNameLike'"
  Exit 0
}

$uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }

$exePath = $null
if ($uninstallString -match '^"([^"]+)"') {
  $exePath = $matches[1]
} elseif ($uninstallString -match '^(.+?\.exe)') {
  $exePath = $matches[1]
}

if (-not $exePath) {
  Write-Host "Could not parse uninstall executable from: $uninstallString"
  Exit 1
}

$installDir = $key.InstallLocation
if (-not $installDir -or -not (Test-Path $installDir)) {
  $installDir = Split-Path -Parent $exePath
}

# NSIS / electron-builder silent uninstall
$argumentList = @("/S", "_?=$installDir")

$process = Start-Process -FilePath $exePath -ArgumentList $argumentList -Wait -PassThru -NoNewWindow
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($installDir -and (Test-Path $installDir)) {
  Remove-Item -Path $installDir -Recurse -Force -ErrorAction SilentlyContinue
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
