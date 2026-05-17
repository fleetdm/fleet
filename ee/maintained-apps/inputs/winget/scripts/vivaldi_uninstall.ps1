# Uninstall Vivaldi (user-scoped Chromium-based browser)

$displayName = "Vivaldi"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and $_.DisplayName -eq $displayName
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found for '$displayName'"
  Exit 1
}

# Kill any running Vivaldi processes before uninstalling
Stop-Process -Name "vivaldi" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$uninstallCommand = if ($uninstall.QuietUninstallString) {
  $uninstall.QuietUninstallString
} else {
  $uninstall.UninstallString
}

# Parse quoted executable from any trailing args in the registry string
$splitArgs = $uninstallCommand.Split('"')
$exe = $splitArgs[1]
$existingArgs = if ($splitArgs.Length -eq 3) { $splitArgs[2].Trim() } else { "" }

# Chromium-based uninstaller flags
$uninstallArgs = "$existingArgs --uninstall --force-uninstall".Trim()

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $uninstallArgs"

try {
  $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -NoNewWindow -PassThru -Wait
  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"

  # Chromium uninstallers return 19 on success
  if ($exitCode -eq 0 -or $exitCode -eq 19) {
    Exit 0
  }
  Exit $exitCode
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
