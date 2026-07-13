# Uninstalls Foxit PDF Editor.
#
# Foxit PDF Editor is a Foxit WiX MultiLangBootstrapper install that runs an auto-updater
# service, so a plain uninstall can be blocked by running processes. Stop the
# Foxit services/processes first, then run the ARP UninstallString (a cached
# burn bundle exe with /uninstall /quiet, or an MsiExec /X{ProductCode}). The
# ARP entry lives in the WOW6432Node (32-bit) hive even on x64.

$softwareName = "Foxit PDF Editor"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$successCodes = @(0, 3010, 1641)
$exitCode = $null

try {

foreach ($svc in @("FoxitPhantomPDFUpdateService")) {
    Stop-Service -Name $svc -Force -ErrorAction SilentlyContinue
}
foreach ($p in @("FoxitPDFEditor", "FoxitPhantomPDF", "FoxitUpdater")) {
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

$raw = $selected.QuietUninstallString
if (-not $raw) { $raw = $selected.UninstallString }
if (-not $raw) { Write-Host "No UninstallString for '$softwareName'."; Exit 1 }

# Parse exe + args (quoted / unquoted / bare).
if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} else {
    $exe = $raw; $exeArgs = ""
}

if ($exe -match '(?i)msiexec') {
    if ($exeArgs -notmatch '(?i)/(x|uninstall)') { $exeArgs = "/X $exeArgs" }
    if ($exeArgs -notmatch '(?i)/(qn|quiet)') { $exeArgs = "$exeArgs /qn" }
    if ($exeArgs -notmatch '(?i)/norestart') { $exeArgs = "$exeArgs /norestart" }
} else {
    if ($exeArgs -notmatch '(?i)/uninstall') { $exeArgs = "/uninstall $exeArgs" }
    if ($exeArgs -notmatch '(?i)/quiet') { $exeArgs = "$exeArgs /quiet" }
    if ($exeArgs -notmatch '(?i)/norestart') { $exeArgs = "$exeArgs /norestart" }
}
$exeArgs = $exeArgs.Trim()

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $exeArgs"
$process = Start-Process -FilePath $exe -ArgumentList $exeArgs -NoNewWindow -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($successCodes -contains $exitCode) { Exit 0 }
Exit $exitCode
