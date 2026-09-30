$softwareName = "SUHelper"
$publisher = "Lenovo"
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

try {
    $key = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $softwareName -and $_.Publisher -eq $publisher } |
        Select-Object -First 1

    if (-not $key) { Write-Host "Uninstall entry not found for '$softwareName'."; Exit 0 }

    $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    }

    Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
} catch { Write-Host "Error: $_"; Exit 1 }

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
