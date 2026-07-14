# Uninstalls Streamlabs Desktop. Its electron-builder ARP DisplayName carries the
# version ("Streamlabs Desktop 1.21.4"), so match by prefix. The app auto-launches
# a tray/updater on install, so stop it first, then run /allusers /S.

$softwareNameLike = "Streamlabs Desktop*"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

foreach ($p in @("Streamlabs Desktop", "Streamlabs OBS", "obs64", "obs32")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

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

# electron-builder NSIS uninstaller: /allusers matches the machine install, /S is silent.
if ($exeArgs -notmatch '(?i)(^|\s)/allusers(\s|$)') { $exeArgs = "$exeArgs /allusers".Trim() }
if ($exeArgs -notmatch '(?i)(^|\s)/S(\s|$)') { $exeArgs = "$exeArgs /S".Trim() }

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
