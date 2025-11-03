# Disable Windows Insider opt-in UI page

# Verify that the registry path exists, and creates it if not.
$regPath = "HKLM:\SOFTWARE\Microsoft\WindowsSelfHost\UI\Visibility"
if (!(Test-Path $regPath)) {
    New-Item -Path $regPath -Force | Out-Null
}
#Set DWord value to hide the Windows insider page under Windows Update
Set-ItemProperty -Path $regPath -Name "HideInsiderPage" -Value 1 -Type DWord
