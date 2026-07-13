# Uninstalls exacqVision Client.
#
# The default upgrade-code MSI uninstall returned a success code but left the
# product registered, because the just-installed client's processes/services
# hold files and msiexec rolls the removal back. So we stop anything running
# from the install dir first, then uninstall by the ProductCode looked up from
# the registry (DisplayName "exacqVision Client (x64)"), and verify it's gone.

$softwareName = "exacqVision Client (x64)"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# 0 = success; 3010/1641 = success but reboot required.
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

# Stop exacqVision processes/services so the MSI uninstall isn't rolled back by
# locked files. Best-effort: everything running from the install location, plus
# known service/process names.
if ($selected.InstallLocation -and (Test-Path -LiteralPath $selected.InstallLocation)) {
    $loc = $selected.InstallLocation.TrimEnd('\')
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

# ProductCode is the uninstall subkey name for an MSI.
$productCode = $selected.PSChildName
if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
    # Fall back to parsing the UninstallString.
    $raw = $selected.UninstallString
    if ($raw -match '(\{[0-9A-Fa-f-]+\})') { $productCode = $matches[1] }
}
if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
    Write-Host "Could not determine ProductCode for '$softwareName'."
    Exit 1
}

Write-Host "Uninstalling product code: $productCode"
$process = Start-Process -FilePath "msiexec.exe" `
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
