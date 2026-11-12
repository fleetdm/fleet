$timeoutSeconds = 300  # 5 minute timeout

# Match only Slack (published by Slack Technologies). We deliberately do NOT rely
# on $PACKAGE_ID or on a PackageFamilyName property: Get-AppxProvisionedPackage
# objects don't expose PackageFamilyName, so an "-eq" match against it is $null on
# every package and would select unrelated packages (e.g. DesktopAppInstaller).
function ShouldRemoveSlackPackage {
  param([Parameter(Mandatory=$true)]$pkg)
  try {
    $name = [string]$pkg.Name
    $family = [string]$pkg.PackageFamilyName
    $publisher = [string]$pkg.Publisher

    if ($name -and ($name -like "*Slack*")) { return $true }
    if ($family -and ($family -like "*Slack*")) { return $true }
    if ($publisher -and ($publisher -like "*Slack Technologies*") -and $name -and ($name -like "*Slack*")) { return $true }
  } catch {}
  return $false
}

try {

  $start = Get-Date

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object {
    ($_.DisplayName -and ($_.DisplayName -like "*Slack*")) -or
    ($_.PackageName -and ($_.PackageName -like "*Slack*"))
  }
  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue | Where-Object {
    ShouldRemoveSlackPackage $_
  }
  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
