# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# WinRAR ships as a self-extracting installer. "-s1" performs a silent
# install with no windows; it installs per-machine into Program Files.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 300

function Stop-ProcessTree {
    param([int]$ParentId)
    Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree -ParentId $_.ProcessId }
    Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
}

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # With WinRAR open, Setup waits on a prompt that can't be answered from
    # SYSTEM and never exits. Stop WinRAR and the tools in its install
    # directory first. Unsaved work in WinRAR is lost.
    $installDirs = @("${env:ProgramFiles}\WinRAR\")
    foreach ($p in @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )) {
        Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like "WinRAR*" -and $_.Publisher -like "win.rar GmbH*"
        } | ForEach-Object {
            $location = $_.InstallLocation
            if (-not $location -and $_.UninstallString -match '^\s*"?([^"]+?)\\[^\\"]+\.exe') {
                $location = $matches[1]
            }
            if ($location) { $installDirs += $location.Trim().Trim('"').TrimEnd('\') + '\' }
        }
    }

    $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $path = $_.Path
        $_.Name -eq 'WinRAR' -or
            ($path -and ($installDirs | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) }))
    })
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

    $process = Start-Process -FilePath "$exeFilePath" -ArgumentList "-s1" -PassThru -NoNewWindow
    # Reading Handle keeps ExitCode available after WaitForExit(ms).
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Stop-ProcessTree -ParentId $process.Id
        Write-Host "Error: WinRAR Setup didn't finish within $($timeoutSeconds / 60) minutes and was stopped. It may be waiting on a prompt, for example because WinRAR was reopened during the install."
        Exit 1
    }
    $process.WaitForExit()

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if ($null -eq $exitCode -or $exitCode -gt [int]::MaxValue) { $exitCode = 1 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
