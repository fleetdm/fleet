# script: Set_ScreenSaverGracePeriod.ps1 
# author: brock@fleetdm.com © 2025 Fleet Device Management
# description: PowerShell command to set MSS:(ScreenSaverGracePeriod) - the time in seconds before the screen saver grace period expires

# Value to be configured for grace period as an interger in seconds
$NumberOfSeconds = "5"

# Command to set value in Windows Registry
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name ScreenSaverGracePeriod -Value $NumberOfSeconds -Force
