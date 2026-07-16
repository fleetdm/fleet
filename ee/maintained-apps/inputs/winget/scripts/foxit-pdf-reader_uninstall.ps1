# Uninstalls Foxit PDF Reader.
#
# Foxit PDF Reader is a Foxit WiX MultiLangBootstrapper install that runs an auto-updater
# service; its visible ARP entry is the inner MSI (its UninstallString uses
# "MsiExec.exe /I{ProductCode}", the maintenance/repair form). We stop the
# Foxit services/processes first so files aren't locked, then resolve the
# ProductCode GUID and run a clean "msiexec /x {ProductCode} /qn /norestart"
# (never reuse the /I from the registry string). The ARP entry lives in the
# WOW6432Node (32-bit) hive even on x64.

$softwareName = "Foxit PDF Reader"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$successCodes = @(0, 3010, 1641)
$exitCode = $null

try {

foreach ($svc in @("FoxitPDFReaderUpdateService")) {
    Stop-Service -Name $svc -Force -ErrorAction SilentlyContinue
}
foreach ($p in @("FoxitPDFReader", "FoxitReader", "FoxitUpdater")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 3

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $uninstallKeys | Where-Object { $_.DisplayName -eq $softwareName } | Select-Object -First 1
if (-not $selected) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

# ProductCode: prefer the ARP key name (it is the MSI ProductCode), else pull
# the first GUID out of the UninstallString. Never carry over its /I switch.
$productCode = $selected.PSChildName
if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
    $raw = $selected.UninstallString
    if ($raw -match '(\{[0-9A-Fa-f-]+\})') { $productCode = $matches[1] }
}
if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
    Write-Host "Could not determine ProductCode for '$softwareName'."
    Exit 1
}

Write-Host "Uninstalling product code: $productCode"
$process = Start-Process msiexec.exe `
    -ArgumentList "/x $productCode /qn /norestart" `
    -NoNewWindow -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($successCodes -contains $exitCode) { Exit 0 }
Exit $exitCode
