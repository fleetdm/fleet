$softwareName = "SUHelper"
$publisher = "Lenovo"
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-SUHelperEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "$softwareName *" -and $_.Publisher -eq $publisher } |
        Select-Object -First 1
}

try {
    $key = Get-SUHelperEntry

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

    # The Inno uninstaller relaunches itself from %TEMP%, so the process above exits before removal finishes.
    $elapsed = 0
    while ((Get-SUHelperEntry) -and ($elapsed -lt 120)) {
        Start-Sleep -Seconds 3
        $elapsed += 3
    }
    if (Get-SUHelperEntry) { Write-Host "SUHelper is still registered after ${elapsed}s."; Exit 1 }
} catch { Write-Host "Error: $_"; Exit 1 }

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
