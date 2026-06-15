# Uninstall script for WiX burn bundle installer.
$softwareName = "Tableau Prep Builder"
$softwareNameLike = "*$softwareName*"
$publisher = "Tableau"

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

    Write-Host "Uninstaller: $uninstallExe"

    $process = Start-Process -FilePath $uninstallExe -ArgumentList @("/uninstall", "/quiet", "/norestart") -PassThru -Wait -NoNewWindow
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
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
