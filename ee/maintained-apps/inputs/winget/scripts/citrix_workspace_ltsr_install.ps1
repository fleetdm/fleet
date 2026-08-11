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
# installer itself. Start without -Wait and poll Programs and Features
# instead of trusting the installer process's exit code.
#
# The bootstrap installs several components as separate, sequential MSI
# transactions (confirmed by CI: "Citrix Workspace Inside" registers before
# "Citrix Workspace(USB)" does). Declaring success as soon as the FIRST
# entry appears races the still-running later components -- our uninstall
# script would then only find and remove whichever ones had registered so
# far. Require the core "Citrix Workspace Inside" entry AND no msiexec.exe
# process running, stable across two consecutive polls, before declaring the
# install done. Match that DisplayName exactly (not a "Citrix Workspace*"
# wildcard) so a leftover entry from an unrelated prior install can't cause
# a false-positive success before this install has actually registered.

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
