# Uninstalls every QGIS LTR (3.44.x) release on the host, since each release
# installs side by side under its own product code. Other QGIS releases are
# left in place.

$ltrSeries = '3.44.'

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED, 1605 = already uninstalled.
$successCodes = @(0, 3010, 1641, 1605)

try {

$uninstallRoots = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)
$entries = @(Get-ItemProperty -Path $uninstallRoots -ErrorAction SilentlyContinue | Where-Object {
  $_.DisplayName -like 'QGIS *' -and $_.Publisher -eq 'QGIS.org' -and
  ([string]$_.DisplayVersion).StartsWith($ltrSeries) -and
  $_.PSChildName -match '^\{[0-9A-Fa-f-]+\}$'
})

if ($entries.Count -eq 0) {
  Write-Host "No QGIS LTR installation found; nothing to uninstall."
  Exit 0
}

$exitCode = 0
foreach ($entry in $entries) {
  Write-Host "Uninstalling $($entry.DisplayName) ($($entry.PSChildName))"
  $process = Start-Process msiexec.exe -ArgumentList "/x $($entry.PSChildName) /qn /norestart" -PassThru -Wait
  Write-Host "Uninstall exit code: $($process.ExitCode)"
  if ($successCodes -notcontains $process.ExitCode) {
    $exitCode = $process.ExitCode
  }
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
