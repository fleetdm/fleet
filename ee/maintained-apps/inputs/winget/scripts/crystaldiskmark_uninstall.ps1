# Locates CrystalDiskMark's Inno Setup uninstaller in the registry and runs it silently.

$softwareNameLike = "CrystalDiskMark *"
$publisher = "Crystal Dew World"
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
$removalTimeoutSeconds = 180

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

function Get-CrystalDiskMarkEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like $softwareNameLike -and $_.Publisher -like "$publisher*" } |
        Select-Object -First 1
}

try {
    $key = Get-CrystalDiskMarkEntry
    if (-not $key -or -not $key.UninstallString) {
        Write-Host "Uninstall entry not found for '$softwareNameLike'."
        Exit 0
    }

    # The uninstaller refuses to run while the benchmark holds its mutex.
    foreach ($name in @("DiskMark32", "DiskMark32A", "DiskMark32M", "DiskMark32S",
                        "DiskMark64", "DiskMark64A", "DiskMark64M", "DiskMark64S",
                        "DiskMarkA64", "DiskMarkA64A", "DiskMarkA64M", "DiskMarkA64S")) {
        Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
    }

    $uninstallCommand = $key.UninstallString
    # Inno quotes the path, but parse the unquoted and bare forms defensively too.
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    }

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -NoNewWindow -PassThru
    $null = $process.Handle
    if (-not $process.WaitForExit($removalTimeoutSeconds * 1000)) {
        Write-Host "Uninstaller process did not exit within ${removalTimeoutSeconds}s, stopping it."
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    }
    if ($process.HasExited) {
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
    }

    # The Inno uninstaller relaunches itself from a temp copy and returns early,
    # so the registry entry disappearing is the real completion signal.
    $elapsed = 0
    while ((Get-CrystalDiskMarkEntry) -and ($elapsed -lt $removalTimeoutSeconds)) {
        Start-Sleep -Seconds 5
        $elapsed += 5
    }

    if (Get-CrystalDiskMarkEntry) {
        Write-Host "CrystalDiskMark is still registered after ${removalTimeoutSeconds}s."
        Exit 1
    }
} catch {
    Write-Host "Error: $_"
    Exit 1
}

Exit $exitCode
