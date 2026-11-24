# Uninstalls the machine-wide Cato Browser with its Chromium uninstaller.
# --force-uninstall skips the confirmation prompt and closes any open browser
# windows. Browsing data in each user's profile is left in place. A copy that a
# user installed only for their own account isn't removed.

$displayName = "Cato Browser"
$successCodes = @(0, 19)  # 19 means the uninstall succeeded

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-CatoBrowserEntry {
    Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $displayName -and $_.UninstallString } |
        Select-Object -First 1
}

try {
    $entry = Get-CatoBrowserEntry
    if (-not $entry) {
        Write-Host "'$displayName' isn't installed for all users. Nothing to uninstall."
        Exit 0
    }

    $uninstallCommand = $entry.UninstallString
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $exe = $Matches[1]
        $uninstallArgs = $Matches[2].Trim()
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $exe = $Matches[1]
        $uninstallArgs = $Matches[2].Trim()
    } else {
        Write-Host "Unable to parse uninstall command: $uninstallCommand"
        Exit 1
    }

    foreach ($flag in @('--uninstall', '--system-level', '--force-uninstall')) {
        if ($uninstallArgs -notmatch "(^|\s)$([regex]::Escape($flag))(\s|$)") {
            $uninstallArgs = "$uninstallArgs $flag".Trim()
        }
    }

    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $uninstallArgs"

    $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if ($successCodes -notcontains $exitCode) {
        if ($null -eq $exitCode -or $exitCode -gt [int]::MaxValue -or $exitCode -lt 0) { $exitCode = 1 }
        Exit $exitCode
    }

    # The uninstaller can exit before its Programs and Features entry is gone.
    $sw = [Diagnostics.Stopwatch]::StartNew()
    while ((Get-CatoBrowserEntry) -and $sw.Elapsed.TotalSeconds -lt 60) {
        Start-Sleep -Seconds 2
    }
    if (Get-CatoBrowserEntry) {
        Write-Host "'$displayName' is still registered after the uninstaller finished."
        Exit 1
    }
    Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
