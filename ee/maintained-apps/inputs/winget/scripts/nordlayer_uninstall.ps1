# Uninstall script for NordLayer (InstallShield/burn EXE wrapping an MSI)
$softwareName = "NordLayer"
$publisher = "NordLayer"

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
      ($_.DisplayName -eq $softwareName -or $_.DisplayName -like "$softwareName*") -and
      ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
    } | Select-Object -First 1
  if ($key) { break }
}

if (-not $key) {
  Write-Host "Uninstall entry not found for '$softwareName'"
  Exit 0
}

# Prefer QuietUninstallString for burn/InstallShield bundles
if ($key.QuietUninstallString) {
  $uninstallString = $key.QuietUninstallString
  $useDefaultArgs = $false
} else {
  $uninstallString = $key.UninstallString
  $useDefaultArgs = $true
}

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

$processOptions = @{
  FilePath    = $exePath
  Wait        = $true
  PassThru    = $true
  NoNewWindow = $true
}
if ($useDefaultArgs) {
  $processOptions.ArgumentList = @("/uninstall", "/quiet", "/norestart")
}

$process = Start-Process @processOptions
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
