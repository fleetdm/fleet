$timeoutSeconds = 300  # 5 minute timeout

# Match only the Nightly channel: its MSIX identity "Mozilla.MozillaFirefoxNightly"
# cannot collide with other Firefox channels' identities. Don't match on a
# PackageFamilyName property: Get-AppxProvisionedPackage doesn't expose it, so an
# "-eq" match is $null for every package.
function ShouldRemoveFirefoxNightlyPackage {
  param([Parameter(Mandatory=$true)]$pkg)
  try {
    $name = [string]$pkg.Name
    $family = [string]$pkg.PackageFamilyName

    if ($name -and ($name -like "*MozillaFirefoxNightly*")) { return $true }
    if ($family -and ($family -like "*MozillaFirefoxNightly*")) { return $true }
  } catch {}
  return $false
}

try {

  $start = Get-Date

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object {
    ($_.DisplayName -and ($_.DisplayName -like "*MozillaFirefoxNightly*")) -or
    ($_.PackageName -and ($_.PackageName -like "*MozillaFirefoxNightly*"))
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
    ShouldRemoveFirefoxNightlyPackage $_
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
