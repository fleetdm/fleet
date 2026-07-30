# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-uninstall-scripts

$softwareName = "OpenRefine"
$publisherMatch = "OpenRefine"

try {
  $uninstallKeys = @(
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKCU:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
  )

  $entries = Get-ChildItem -Path $uninstallKeys -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath } |
    Where-Object {
      ($_.DisplayName -eq $softwareName -or $_.DisplayName -like "$softwareName*") -and
      ($publisherMatch -eq "" -or $_.Publisher -like "*$publisherMatch*")
    }

  if (-not $entries) {
    Write-Host "Uninstall entry not found for '$softwareName'"
    Exit 0
  }

  $key = $entries | Select-Object -First 1
  $uninstallString = $key.UninstallString
  if (-not $uninstallString) {
    Write-Host "No UninstallString found for '$softwareName'"
    Exit 0
  }

  # Parse the executable path from the UninstallString
  if ($uninstallString -match '^"([^"]+)"') {
    $exePath = $matches[1]
  } elseif ($uninstallString -match '^(.+?\.exe)') {
    $exePath = $matches[1]
  } else {
    $exePath = $uninstallString
  }

  # Determine install directory
  $installDir = $key.InstallLocation
  if ([string]::IsNullOrWhiteSpace($installDir) -or -not (Test-Path $installDir)) {
    $installDir = Split-Path -Parent $exePath
  }

  $argumentList = @("/VERYSILENT", "/NORESTART")

  Write-Host "Uninstall exe: $exePath"
  Write-Host "Uninstall args: $argumentList"

  $process = Start-Process -FilePath $exePath -ArgumentList $argumentList -Wait -PassThru -NoNewWindow
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"

  if ($installDir -and (Test-Path $installDir)) {
    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
  }

  Exit $exitCode
} catch {
  Write-Host "Error: $_"
  Exit 1
}
