# Uninstalls exacqVision Client.
#
# The default upgrade-code uninstall found the product but returned success
# while leaving it registered, because the just-installed client's processes
# hold files and msiexec rolls the removal back. So we stop anything running
# from the install dir (plus known process/service names) FIRST, then resolve
# the ProductCode(s) via the stable UpgradeCode and msiexec /x each.

$upgradeCode = '{9F63FCC6-07FA-48B8-A4B0-F31364C18DAF}'
$softwareName = 'exacqVision Client (x64)'
$successCodes = @(0, 3010, 1641)

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

try {

# Stop exacqVision processes/services so the MSI uninstall isn't rolled back by
# locked files.
$entry = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -eq $softwareName } | Select-Object -First 1
if ($entry -and $entry.InstallLocation -and (Test-Path -LiteralPath $entry.InstallLocation)) {
    $loc = $entry.InstallLocation.TrimEnd('\')
    Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}
foreach ($p in @("exacqVisionClient", "evClient", "exacqVision", "edvrService")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}
foreach ($svc in @("edvrserver", "exacqVisionServer")) {
    Stop-Service -Name $svc -Force -ErrorAction SilentlyContinue
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
