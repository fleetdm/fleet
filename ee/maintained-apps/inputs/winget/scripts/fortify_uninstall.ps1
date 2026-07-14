# Uninstalls Fortify.
#
# Fortify auto-starts a tray process (no Windows service), which can hold files
# and make msiexec roll back the uninstall. Stop anything running from the
# install dir (plus known process names) FIRST, then resolve the ProductCode(s)
# via the stable UpgradeCode and msiexec /x each.

$upgradeCode = '{AB87B5E7-17F6-5394-8A15-9EE6AA6B06B8}'
$softwareName = 'Fortify'
$successCodes = @(0, 3010, 1641)

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

try {

# Stop the app's processes so the MSI uninstall isn't rolled back by locked files.
$entry = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -eq $softwareName } | Select-Object -First 1
if ($entry -and $entry.InstallLocation -and (Test-Path -LiteralPath $entry.InstallLocation)) {
    $loc = $entry.InstallLocation.TrimEnd('\')
    Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}
foreach ($p in @("Fortify")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 3

# Resolve ProductCode(s) from the stable UpgradeCode and uninstall each.
$inst = New-Object -ComObject "WindowsInstaller.Installer"
$productCodes = @($inst.RelatedProducts($upgradeCode))
if ($productCodes.Count -eq 0) {
    Write-Host "No products found for upgrade code $upgradeCode"
    Exit 1
}

foreach ($productCode in $productCodes) {
    Write-Host "Uninstalling product code: $productCode"
    $process = Start-Process msiexec.exe `
        -ArgumentList "/x $productCode /quiet /norestart" `
        -NoNewWindow -PassThru -Wait
    Write-Host "Uninstall exit code: $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) { Exit $process.ExitCode }
}

Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
