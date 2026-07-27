# Uninstalls HP Prime Virtual Calculator (WiX Burn bundle). Runs the cached
# bundle's QuietUninstallString (/uninstall /quiet).

$softwareName = "HP Prime Virtual Calculator"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $uninstallKeys | Where-Object { $_.DisplayName -eq $softwareName } | Select-Object -First 1
if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

$raw = if ($selected.QuietUninstallString) { $selected.QuietUninstallString } else { $selected.UninstallString }
if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} else {
    $exe = $raw; $exeArgs = ""
}
if ($exeArgs -notmatch '(?i)(^|\s)/uninstall(\s|$)') { $exeArgs = "/uninstall $exeArgs".Trim() }
if ($exeArgs -notmatch '(?i)(^|\s)/quiet(\s|$)') { $exeArgs = "$exeArgs /quiet".Trim() }
if ($exeArgs -notmatch '(?i)/norestart') { $exeArgs = "$exeArgs /norestart".Trim() }

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $exeArgs"
$process = Start-Process -FilePath $exe -ArgumentList $exeArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($exitCode -eq 0 -or $exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
