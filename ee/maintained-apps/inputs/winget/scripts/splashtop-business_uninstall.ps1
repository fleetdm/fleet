# Splashtop Business is installed by an MSI, so it's removed with msiexec.
# Uninstalling closes a running Splashtop Business and also deletes its
# settings and logs for every user on the host.

$softwareName = "Splashtop Business"
$publisher = "Splashtop Inc."
$successCodes = @(0, 3010, 1641)
$timeoutSeconds = 900

$uninstallRoots = @(
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

# The uninstall key of an MSI product is named after its product code.
function Get-InstalledProductCodes {
  foreach ($root in $uninstallRoots) {
    Get-ChildItem -LiteralPath $root -ErrorAction SilentlyContinue | ForEach-Object {
      $item = Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction SilentlyContinue
      if ($item -and $item.DisplayName -eq $softwareName -and $item.Publisher -eq $publisher -and
          $item.PSChildName -match '^\{[0-9A-Fa-f-]{36}\}$') {
        $item.PSChildName
      }
    }
  }
}

try {

$productCodes = @(Get-InstalledProductCodes | Select-Object -Unique)
if ($productCodes.Count -eq 0) {
  Write-Host "$softwareName uninstall entry not found; nothing to do."
  Exit 0
}

foreach ($productCode in $productCodes) {
  Write-Host "Uninstalling product code: $productCode"
  $process = Start-Process -FilePath "msiexec.exe" `
    -ArgumentList "/x $productCode /qn /norestart" -PassThru -NoNewWindow
  # Keeps .ExitCode readable after the process ends.
  $null = $process.Handle

  if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
    Write-Host "msiexec did not exit within $timeoutSeconds seconds; terminating it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
    Exit 1603
  }

  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
  if ($successCodes -notcontains $exitCode) { Exit $exitCode }
}

if (@(Get-InstalledProductCodes).Count -gt 0) {
  Write-Host "$softwareName is still registered after the uninstall."
  Exit 1
}

Write-Host "$softwareName is no longer present."
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
