# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Raspberry Pi Imager uses an Inno Setup installer; these switches run it
# silently and it installs machine-wide when run elevated. Despite its [Run]
# entry being flagged skipifsilent, the Setup process does not exit on its
# own after a silent install finishes -- it keeps running and holds the
# installer file lock indefinitely. A plain "Start-Process -Wait" would
# therefore block until killed even though the files install correctly. So
# this launches Setup, polls until Raspberry Pi Imager is registered in
# Programs and Features, then stops the lingering process to release the
# lock.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 300
$pollIntervalSeconds = 5

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Test-RaspberryPiImagerInstalled {
    try {
        $props = Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -eq 'Raspberry Pi Imager' } |
            Select-Object -First 1
        return [bool]$props
    } catch {
        return $false
    }
}

# Recursively stop the Setup process and any children (e.g. Inno's setup.tmp
# helper) so the installer file lock is released.
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

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Launched Raspberry Pi Imager installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        if (Test-RaspberryPiImagerInstalled) {
            Write-Host "Raspberry Pi Imager registered in Programs and Features after ${elapsed}s"
            if (-not $process.HasExited) {
                Stop-ProcessTree -ParentId $process.Id
                Write-Host "Stopped lingering installer process to release file lock"
            }
            Exit 0
        }

        # If Setup exits on its own, trust its exit code (after a final check).
        if ($process.HasExited) {
            Start-Sleep -Seconds 2
            if (Test-RaspberryPiImagerInstalled) { Exit 0 }
            $exitCode = $process.ExitCode
            Write-Host "Installer exited with code $exitCode but Raspberry Pi Imager was not detected"
            # 3010 = success, reboot required.
            if ($exitCode -eq 3010) { Exit 0 }
            Exit $exitCode
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (Test-RaspberryPiImagerInstalled) {
        if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for Raspberry Pi Imager to register in Programs and Features"
    if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
