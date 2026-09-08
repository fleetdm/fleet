# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Raspberry Pi Imager uses an Inno Setup installer; it needs /VERYSILENT to
# run silently and installs machine-wide when run elevated. Its installer
# stays running after a silent install, so this script waits for Raspberry
# Pi Imager to register in Programs and Features, then stops it.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 300
$pollIntervalSeconds = 5

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

# Returns the registered DisplayVersion for Raspberry Pi Imager, or $null if
# not currently installed.
function Get-RaspberryPiImagerVersion {
    try {
        $props = Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -eq 'Raspberry Pi Imager' } |
            Select-Object -First 1
        if ($props) { return $props.DisplayVersion }
        return $null
    } catch {
        return $null
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

    $beforeVersion = Get-RaspberryPiImagerVersion

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Launched Raspberry Pi Imager installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        $currentVersion = Get-RaspberryPiImagerVersion
        if ($currentVersion -and $currentVersion -ne $beforeVersion) {
            Write-Host "Raspberry Pi Imager $currentVersion registered in Programs and Features after ${elapsed}s"
            if (-not $process.HasExited) {
                Stop-ProcessTree -ParentId $process.Id
                Write-Host "Stopped lingering installer process to release file lock"
            }
            Exit 0
        }

        if ($process.HasExited) {
            $exitCode = $process.ExitCode
            if ($exitCode -ne 0 -and $exitCode -ne 3010) {
                Write-Host "Installer exited with code $exitCode"
                Exit $exitCode
            }
            Start-Sleep -Seconds 2
            if (Get-RaspberryPiImagerVersion) { Exit 0 }
            Write-Host "Installer exited with code $exitCode but Raspberry Pi Imager was not detected"
            Exit 1
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    # Setup is still "running" per Windows, but it's had the full budget to
    # finish its work, so presence alone is trusted here.
    if (Get-RaspberryPiImagerVersion) {
        Write-Host "Raspberry Pi Imager present after ${pollTimeoutSeconds}s; treating as complete"
        if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for Raspberry Pi Imager to be detected"
    if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
