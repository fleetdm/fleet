$softwareNameLike = "HandBrake *"
$uninstallArgs = "/S"
$removalTimeoutSeconds = 180

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

function Get-HandBrakeEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like $softwareNameLike } |
        Select-Object -First 1
}

try {
    $key = Get-HandBrakeEntry
    if (-not $key) { Write-Host "Uninstall entry not found for '$softwareNameLike'."; Exit 0 }

    $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    # HandBrake writes an unquoted path that contains spaces, so capture through .exe.
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    }

    Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    $null = $process.Handle
    $null = $process.WaitForExit($removalTimeoutSeconds * 1000)
    if ($process.HasExited) { $exitCode = $process.ExitCode; Write-Host "Uninstall exit code: $exitCode" }

    # A silent NSIS uninstaller returns before removal finishes, so the registry
    # entry disappearing is the real completion signal.
    $elapsed = 0
    while ((Get-HandBrakeEntry) -and ($elapsed -lt $removalTimeoutSeconds)) {
        Start-Sleep -Seconds 5
        $elapsed += 5
    }

    if (Get-HandBrakeEntry) { Write-Host "HandBrake is still registered after ${removalTimeoutSeconds}s."; Exit 1 }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit $exitCode
