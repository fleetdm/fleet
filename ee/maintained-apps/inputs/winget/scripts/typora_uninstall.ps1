# Uninstall script for Inno Setup installer.
$softwareName = "Typora"
$softwareNameLike = "*$softwareName*"
$publisher = ""

$keys = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKCU:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem -Path $keys -ErrorAction SilentlyContinue |
  ForEach-Object { Get-ItemProperty $_.PSPath }

$found = $false
foreach ($key in $uninstallKeys) {
  if ($key.DisplayName -like $softwareNameLike -and ($publisher -eq "" -or $key.Publisher -like "*$publisher*")) {
    $found = $true

    $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }

    if ($uninstallCommand -match '^"([^"]+)"') {
      $uninstallExe = $matches[1]
    } elseif ($uninstallCommand -match '^(.+?\.exe)') {
      $uninstallExe = $matches[1]
    } else {
      $uninstallExe = $uninstallCommand
    }

    if ($key.InstallLocation -and (Test-Path $key.InstallLocation)) {
      $installDir = $key.InstallLocation
    } else {
      $installDir = Split-Path -Parent $uninstallExe
    }

    Write-Host "Uninstaller: $uninstallExe"
    Write-Host "Install dir: $installDir"

    $process = Start-Process -FilePath $uninstallExe -ArgumentList @("/VERYSILENT", "/NORESTART") -PassThru -Wait -NoNewWindow
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
    break
  }
}

if (-not $found) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

} catch {
  Write-Host "Error: $_"
  Exit 1
}

Exit $exitCode
