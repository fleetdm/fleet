# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Vendor-documented silent switches (winget InstallerSwitches: Silent +
# Custom): /silent runs without dialogs/prompts, /noreboot suppresses the
# reboot prompt, and /AutoUpdateCheck=disabled keeps Citrix's own updater
# from taking over version management.
#
# The Citrix bootstrapper leaves resident processes running after install
# (e.g. its self-service/notification tray app), so Start-Process -Wait
# never returns -- it waits on the whole process tree, not just the
# installer itself. Start without -Wait and poll Programs and Features for
# the app's own registry entry instead of trusting the installer process's
# exit code.

$softwareNameLike = "Citrix Workspace *"
$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$timeoutSeconds = 480
$pollIntervalSeconds = 10

function Test-CitrixWorkspaceInstalled {
  [array]$uninstallKeys = Get-ChildItem `
      -Path $paths `
      -ErrorAction SilentlyContinue |
          ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

  foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike `
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
while ($elapsed -lt $timeoutSeconds) {
  if (Test-CitrixWorkspaceInstalled) {
    Write-Host "Citrix Workspace registered in Programs and Features after ${elapsed}s"
    Exit 0
  }
  Start-Sleep -Seconds $pollIntervalSeconds
  $elapsed += $pollIntervalSeconds
}

Write-Host "Timed out after ${timeoutSeconds}s waiting for Citrix Workspace to register"
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
