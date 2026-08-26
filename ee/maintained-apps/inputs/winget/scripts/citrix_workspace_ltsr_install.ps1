# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# The bootstrap installs components as separate MSI transactions, so -Wait
# can't be used; poll for the core entry and for msiexec to go idle.

$softwareName = "Citrix Workspace Inside"
$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$timeoutSeconds = 480
$pollIntervalSeconds = 10
$requiredStableChecks = 2

function Test-CitrixWorkspaceInstalled {
  [array]$uninstallKeys = Get-ChildItem `
      -Path $paths `
      -ErrorAction SilentlyContinue |
          ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

  foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareName `
        -and $key.Publisher -eq "Citrix Systems, Inc.") {
      return $true
    }
  }
  return $false
}

try {

Start-Process -FilePath "${env:INSTALLER_PATH}" `
  -ArgumentList "/silent /noreboot /AutoUpdateCheck=disabled" `
  -PassThru | Out-Null

$elapsed = 0
$stableChecks = 0
while ($elapsed -lt $timeoutSeconds) {
  $registered = Test-CitrixWorkspaceInstalled
  $msiexecIdle = -not (Get-Process -Name "msiexec" -ErrorAction SilentlyContinue)

  if ($registered -and $msiexecIdle) {
    $stableChecks++
    if ($stableChecks -ge $requiredStableChecks) {
      Write-Host "Citrix Workspace registered and no MSI transaction in flight after ${elapsed}s"
      Exit 0
    }
  } else {
    $stableChecks = 0
  }

  Start-Sleep -Seconds $pollIntervalSeconds
  $elapsed += $pollIntervalSeconds
}

Write-Host "Timed out after ${timeoutSeconds}s waiting for Citrix Workspace to finish installing"
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
