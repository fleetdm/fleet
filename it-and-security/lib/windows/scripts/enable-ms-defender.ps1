# Enable Windows Defender
# Based on commands found here: https://support.huntress.io/hc/en-us/articles/4402989131283-Enable-Microsoft-Defender-via-PowerShell
# Enable Real-Time Monitoring
Set-MpPreference -DisableRealtimeMonitoring $false

# Enable IOAV Protection
Set-MpPreference -DisableIOAVProtection $false

# Create Registry Key for Real-Time Protection
New-Item -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender" -Name "Real-Time Protection" -Force

# Enable Behavior Monitoring
New-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection" -Name "DisableBehaviorMonitoring" -Value 0 -PropertyType DWORD -Force

# Enable On-Access Protection
New-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection" -Name "DisableOnAccessProtection" -Value 0 -PropertyType DWORD -Force

# Ensure Scans Run When Real-Time Protection is Enabled
New-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection" -Name "DisableScanOnRealtimeEnable" -Value 0 -PropertyType DWORD -Force

# Ensure AntiSpyware is Enabled
New-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender" -Name "DisableAntiSpyware" -Value 0 -PropertyType DWORD -Force

# Start Windows Defender Services
Start-Service -Name WinDefend
Start-Service -Name WdNisSvc
