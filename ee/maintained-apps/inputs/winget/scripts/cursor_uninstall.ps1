# Attempts to locate Cursor's uninstaller from registry and execute it silently
# If found, adds common silent flags for Inno or MSI uninstallers

$displayName = "Cursor"
$publisher = "Anysphere"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$cmd = $uninstall.UninstallString
if ($cmd -match 'unins.*\\.exe') { $cmd += ' /VERYSILENT /SUPPRESSMSGBOXES /NORESTART' }
elseif ($cmd -match 'msiexec') { if ($cmd -notmatch '/x') { $cmd = $cmd + ' /x' }; $cmd += ' /quiet /norestart' }

try {
  $process = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile","-NonInteractive","-Command", $cmd -PassThru -Wait
  Exit $process.ExitCode
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
