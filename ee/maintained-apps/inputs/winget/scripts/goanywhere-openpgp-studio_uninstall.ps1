# Uninstalls GoAnywhere OpenPGP Studio. install4j registers an ARP entry whose
# UninstallString points at its uninstall.exe; -q runs it unattended.

$softwareNameLike = "GoAnywhere OpenPGP Studio*"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

foreach ($p in @("OpenPGPStudio", "GoAnywhere OpenPGP Studio")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}

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

$raw = $selected.UninstallString
if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $matches[1]; $exeArgs = $matches[2].Trim()
} else {
    $exe = $raw; $exeArgs = ""
}

# install4j silent uninstall flag.
if ($exeArgs -notmatch '(?i)(^|\s)-q(\s|$)') { $exeArgs = "$exeArgs -q".Trim() }

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
