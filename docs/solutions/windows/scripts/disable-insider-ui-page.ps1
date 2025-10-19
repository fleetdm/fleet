# Disable Windows Insider opt-in UI page

$regPath = "HKLM:\SOFTWARE\Microsoft\WindowsSelfHost\UI\Visibility"
if (!(Test-Path $regPath)) {
    New-Item -Path $regPath -Force | Out-Null
}

# Turn off preview builds and prevent joining the Insider Program
Set-ItemProperty -Path $regPath -Name "HideInsiderPage" -Value 1 -Type DWord
