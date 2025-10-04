# Prevents uninstall/change of Fleet osquery via Windows UI.
# Sets NoRemove and NoModify = 1 under Fleet osquery uninstall entry.
# Hides uninstall/change options across Control Panel and Settings > Apps.
# Works on all Windows editions.

$UninstallPaths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)

$FleetEntry = Get-ItemProperty -Path $UninstallPaths -ErrorAction SilentlyContinue |
              Where-Object { $_.DisplayName -like "Fleet osquery*" }

if ($FleetEntry) {
    Write-Output "[INFO] Fleet osquery found: $($FleetEntry.DisplayName)"
    $RegKeyPath = $FleetEntry.PSPath

    New-ItemProperty -Path $RegKeyPath -Name "NoRemove" -Value 1 -PropertyType DWord -Force | Out-Null
    Write-Output "[SET] NoRemove = 1"

    New-ItemProperty -Path $RegKeyPath -Name "NoModify" -Value 1 -PropertyType DWord -Force | Out-Null
    Write-Output "[SET] NoModify = 1"

    Write-Output "[DONE] Fleet osquery uninstall options hardened."
} else {
    Write-Output "[WARN] Fleet osquery not found. Nothing changed."
}
