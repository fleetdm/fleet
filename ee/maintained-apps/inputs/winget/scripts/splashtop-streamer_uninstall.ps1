$softwareName = "Splashtop Streamer"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$successCodes = @(0, 3010, 1641)
$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $uninstallKeys | Where-Object { $_.DisplayName -eq $softwareName } | Select-Object -First 1
if (-not $selected) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

# The ARP key name is the MSI ProductCode; fall back to the GUID in UninstallString.
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
