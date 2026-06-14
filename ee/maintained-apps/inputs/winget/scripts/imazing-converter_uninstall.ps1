# Attempts to locate iMazing HEIC Converter's uninstaller from the registry and execute it silently (Inno Setup)

$displayName = "iMazing HEIC Converter"
$publisher = "DigiDNA"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

try {
  $uninstallCommand = $uninstall.UninstallString

  # Parse the executable from the uninstall string
  $exePath = $null
  if ($uninstallCommand -match '^"([^"]+)"') {
    $exePath = $Matches[1]
  } elseif ($uninstallCommand -match '^(.+?\.exe)') {
    $exePath = $Matches[1]
  }

  if (-not $exePath) {
    Write-Host "Could not parse uninstaller path from: $uninstallCommand"
    Exit 1
  }

  # Determine install directory
  $installDir = $null
  if ($uninstall.InstallLocation -and (Test-Path $uninstall.InstallLocation)) {
    $installDir = $uninstall.InstallLocation
  } else {
    $installDir = Split-Path -Parent $exePath
  }

  Write-Host "Uninstaller: $exePath"

  $processOptions = @{
    FilePath = $exePath
    ArgumentList = @("/VERYSILENT", "/NORESTART")
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode

  # Clean up any leftover install directory
  if ($installDir -and (Test-Path $installDir)) {
    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
  }

  Write-Host "Uninstall exit code: $exitCode"
  Exit $exitCode
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
