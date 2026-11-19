# Removes the ChatGPT MSIX (package OpenAI.Codex) provisioned and registered for every
# user. ChatGPT Classic (OpenAI.ChatGPT-Desktop) is a separate package and is left alone.

$packageName = "OpenAI.Codex"
$packageFamilyName = "OpenAI.Codex_2p2nqsd0c76g0"
$timeoutSeconds = 300

function Get-Provisioned {
  Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object { $_.DisplayName -eq $packageName }
}

function Get-Registered {
  Get-AppxPackage -AllUsers -Name $packageName -ErrorAction Stop |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
}

try {

  $start = Get-Date

  # Match on the package's install folder so a running ChatGPT Classic isn't closed too.
  Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "*\WindowsApps\${packageName}_*" } |
    Stop-Process -Force -ErrorAction SilentlyContinue

  foreach ($pkg in @(Get-Provisioned)) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    try {
      Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    } catch {
      Write-Host "Remove-AppxProvisionedPackage reported: $($_.Exception.Message)"
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { break }
  }

  foreach ($app in @(Get-Registered)) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    try {
      Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    } catch {
      Write-Host "Remove-AppxPackage reported: $($_.Exception.Message)"
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { break }
  }

  # A removal cmdlet can report an error and still remove the package, so check the result.
  $remaining = @(Get-Provisioned) + @(Get-Registered)
  if ($remaining.Count -eq 0) {
    Write-Host "$packageFamilyName is no longer present."
    Exit 0
  }

  Write-Host "$packageFamilyName is still present after removal attempts."
  Exit 1603

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
