# configure-windows-update-registry.ps1
# CIS Windows 11 Enterprise Benchmark v4.0.0 - Windows Update Registry Settings
# Configures Windows Update settings that require direct registry manipulation

$ErrorActionPreference = 'Stop'

$wuPath = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate'
$auPath = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU'

if (-not (Test-Path $wuPath)) { New-Item -Path $wuPath -Force | Out-Null }
if (-not (Test-Path $auPath)) { New-Item -Path $auPath -Force | Out-Null }

# CIS 18.10.93.1.1 - No auto-restart with logged on users = Disabled (value 0)
Set-ItemProperty -Path $auPath -Name 'NoAutoRebootWithLoggedOnUsers' -Value 0 -Type DWord -Force

# WU-OPS-011 - TargetReleaseVersionInfo (set to current target release)
Set-ItemProperty -Path $wuPath -Name 'TargetReleaseVersionInfo' -Value '24H2' -Type String -Force

# WU-OPS-012 - IncludeRecommendedUpdates = 1 (include recommended updates)
Set-ItemProperty -Path $auPath -Name 'IncludeRecommendedUpdates' -Value 1 -Type DWord -Force

Write-Output "Windows Update registry configuration complete."
