$timeoutSeconds = 300  # 5 minute timeout

function ShouldRemoveClaudePackage {
  param([Parameter(Mandatory=$true)]$pkg)
  try {
    $name = [string]$pkg.Name
    $family = [string]$pkg.PackageFamilyName
    $publisher = [string]$pkg.Publisher

    if ($name -and ($name -like "*Claude*" -or $name -like "*Anthropic*")) { return $true }
    if ($family -and ($family -like "*Claude*" -or $family -like "*Anthropic*")) { return $true }
    if ($publisher -and ($publisher -like "*Anthropic*")) { return $true }
  } catch {}
  return $false
}

try {

  $start = Get-Date

  # Best-effort: close app if running (name may vary)
  Stop-Process -Name "Claude" -Force -ErrorAction SilentlyContinue

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object {
    ($_.PackageFamilyName -and (($_.PackageFamilyName -like "*Claude*") -or ($_.PackageFamilyName -like "*Anthropic*"))) -or
    ($_.DisplayName -and (($_.DisplayName -like "*Claude*") -or ($_.DisplayName -like "*Anthropic*"))) -or
    ($_.PackageName -and (($_.PackageName -like "*Claude*") -or ($_.PackageName -like "*Anthropic*")))
  }

  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) { Exit 1603 }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue | Where-Object {
    ShouldRemoveClaudePackage $_
  }

  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) { Exit 1603 }
  }

  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
