# eM Client installs as a small "eM Client (Updater)" MSI plus an eM Client app
# package for each user. Removing the MSI leaves those packages installed, so
# this script removes the MSI and then the package for every user. eM Client is
# closed if it's running.

$upgradeCode = '{D8A1DC6C-8EDA-4135-8E82-6F4EEA116B2F}'
$packageName = 'eMClient.20054CA46072C'
$timeoutSeconds = 300
# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

Get-Process -Name 'MailClient' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

try {
  $inst = New-Object -ComObject 'WindowsInstaller.Installer'
  foreach ($productCode in @($inst.RelatedProducts($upgradeCode))) {
    $process = Start-Process msiexec -ArgumentList @('/quiet', '/x', $productCode, '/norestart') -PassThru
    # Caching the handle keeps ExitCode readable after a timed WaitForExit.
    $null = $process.Handle
    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
      Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
      Write-Host "Uninstall of $productCode timed out"
      Exit 1603
    }
    Write-Host "Uninstall of $productCode exited $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) {
      Exit $process.ExitCode
    }
  }

  Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -eq $packageName } | ForEach-Object {
    Write-Host "Removing provisioned package $($_.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $_.PackageName -AllUsers -ErrorAction Stop | Out-Null
  }

  Get-AppxPackage -AllUsers -Name $packageName | ForEach-Object {
    Write-Host "Removing package $($_.PackageFullName)"
    Remove-AppxPackage -Package $_.PackageFullName -AllUsers -ErrorAction Stop
  }
} catch {
  Write-Host "Error: $_"
  Exit 1603
}

Exit 0
