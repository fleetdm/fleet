# Each QGIS release installs side by side under its own product code, so
# installed QGIS LTR (3.44.x) releases are uninstalled first. Other QGIS
# releases are left in place.

$ltrSeries = '3.44.'
$logFile = "${env:TEMP}/fleet-install-software.log"

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

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
foreach ($entry in $entries) {
  Write-Host "Uninstalling $($entry.DisplayName) ($($entry.PSChildName))"
  $process = Start-Process msiexec.exe -ArgumentList "/x $($entry.PSChildName) /qn /norestart" -PassThru -Wait
  # 1605 = the release was already uninstalled
  if (($successCodes + 1605) -notcontains $process.ExitCode) {
    Write-Host "Uninstall of $($entry.DisplayName) failed with exit code $($process.ExitCode)"
    Exit $process.ExitCode
  }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
