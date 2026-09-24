# Uninstalls IrfanView. Its ARP DisplayName carries the version+arch
# (e.g. "IrfanView 4.75 (64-bit)"), so match by prefix and run the registered
# uninstaller (iv_uninstall.exe) silently.

$softwareNameLike = "IrfanView*"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $uninstallKeys |
    Where-Object { $_.DisplayName -like $softwareNameLike } |
    Select-Object -First 1
if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
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
if ($exeArgs -notmatch '(?i)(^|\s)/silent(\s|$)') { $exeArgs = "$exeArgs /silent".Trim() }

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
