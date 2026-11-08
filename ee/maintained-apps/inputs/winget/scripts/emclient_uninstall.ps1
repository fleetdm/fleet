# eM Client installs as a small "eM Client (Updater)" MSI plus an eM Client app
# package for each user. Removing the MSI leaves those packages installed, so
# this script removes the MSI and then the package for every user, and waits for
# Windows to finish removing it. eM Client is closed if it's running.

$upgradeCode = '{D8A1DC6C-8EDA-4135-8E82-6F4EEA116B2F}'
$packageName = 'eMClient.20054CA46072C'
# Under Fleet's 5-minute script timeout, so a failure exits 1603 instead of being killed.
$timeoutSeconds = 240
# Monotonic, so a clock change can't stretch the wait past Fleet's limit.
$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

# A provisioned package or a signed-in user's copy counts as left behind. A signed-out
# user's copy counts unless Windows marked it "(pending removal)", which finishes at their
# next sign-in. A Staged entry isn't registered for anyone. The queries stop on errors so
# a failed query isn't read as nothing left.
function Test-PackageRemains {
  if (@(Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object { $_.DisplayName -eq $packageName }).Count -gt 0) {
    return $true
  }
  foreach ($package in @(Get-AppxPackage -AllUsers -Name $packageName -ErrorAction Stop)) {
    foreach ($user in @($package.PackageUserInformation)) {
      $state = $user.InstallState
      if ($state -like 'Staged*') {
        continue
      }
      if ($state -notlike '*(pending removal)' -or
          (Test-Path "Registry::HKEY_USERS\$($user.UserSecurityId.Sid)" -ErrorAction Stop)) {
        return $true
      }
    }
  }
  return $false
}

Get-Process -Name 'MailClient' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

try {
  $inst = New-Object -ComObject 'WindowsInstaller.Installer'
  foreach ($productCode in @($inst.RelatedProducts($upgradeCode))) {
    $process = Start-Process msiexec -ArgumentList @('/quiet', '/x', $productCode, '/norestart') -PassThru
    # Caching the handle keeps ExitCode readable after a timed WaitForExit.
    $null = $process.Handle
    $remainingSeconds = [Math]::Max(0, $timeoutSeconds - $stopwatch.Elapsed.TotalSeconds)
    if (-not $process.WaitForExit([int]($remainingSeconds * 1000))) {
      Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
      Write-Host "Uninstall of $productCode timed out"
      Exit 1603
    }
    Write-Host "Uninstall of $productCode exited $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) {
      Exit $process.ExitCode
    }
  }

  # The removal cmdlets can report errors for removals that succeed, and Windows can
  # finish a removal after they return, so wait for what's left to go instead.
  Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object { $_.DisplayName -eq $packageName } | ForEach-Object {
    Write-Host "Removing provisioned package $($_.PackageName)"
    try {
      Remove-AppxProvisionedPackage -Online -PackageName $_.PackageName -AllUsers -ErrorAction Stop | Out-Null
    } catch {
      Write-Host "Remove-AppxProvisionedPackage reported: $($_.Exception.Message)"
    }
  }

  Get-AppxPackage -AllUsers -Name $packageName -ErrorAction Stop | ForEach-Object {
    Write-Host "Removing package $($_.PackageFullName)"
    try {
      Remove-AppxPackage -Package $_.PackageFullName -AllUsers -ErrorAction Stop
    } catch {
      Write-Host "Remove-AppxPackage reported: $($_.Exception.Message)"
    }
  }

  do {
    $checkFailed = $false
    try {
      if (-not (Test-PackageRemains)) {
        Exit 0
      }
    } catch {
      $checkFailed = $true
      Write-Host "Checking for eM Client failed, retrying: $($_.Exception.Message)"
    }
    Start-Sleep -Seconds 5
  } while ($stopwatch.Elapsed.TotalSeconds -lt $timeoutSeconds)

  if ($checkFailed) {
    Write-Host "Couldn't verify that eM Client was removed."
  } else {
    Write-Host "eM Client is still installed after removal."
  }
  Exit 1603
} catch {
  Write-Host "Error: $_"
  Exit 1603
}
