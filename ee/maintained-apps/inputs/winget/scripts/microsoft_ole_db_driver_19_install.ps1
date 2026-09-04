# Requires IACCEPTMSOLEDBSQLLICENSETERMS=YES to install unattended. Any existing
# same-version install is removed first - SecureRepair blocks a same-version
# reinstall and fails with 1603.

$logFile = "${env:TEMP}/fleet-install-software.log"
$softwareName = "Microsoft OLE DB Driver 19 for SQL Server"
$softwarePublisher = "Microsoft Corporation"

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
$successCodes = @(0, 3010, 1641)

try {

$machineKey       = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

[array]$existing = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath } |
    Where-Object { $_.DisplayName -eq $softwareName -and $_.Publisher -eq $softwarePublisher }

foreach ($entry in $existing) {
    $productCode = $entry.PSChildName
    if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
        continue
    }
    $removeProcess = Start-Process msiexec.exe `
        -ArgumentList "/x $productCode /qn /norestart" -NoNewWindow -PassThru -Wait
    if (-not ($successCodes -contains $removeProcess.ExitCode)) {
        Exit $removeProcess.ExitCode
    }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${env:INSTALLER_PATH}`" IACCEPTMSOLEDBSQLLICENSETERMS=YES" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
