# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# darktable ships as an Inno Setup installer (NOT NSIS, despite winget's
# metadata reporting "nullsoft"). Inno ignores the NSIS "/S" switch and launches
# its GUI wizard, so we pass Inno's silent switches instead.
#
# darktable's installer has a postinstall [Run] entry that opens the online user
# manual via shellexec/runasoriginaluser and is NOT flagged "skipifsilent", so it
# fires even under /VERYSILENT. On a headless host that step never returns,
# leaving the Setup process alive indefinitely and holding the installer-file
# lock -- a plain "Start-Process -Wait" would block until killed even though the
# files install correctly. So we launch Setup, poll until darktable is registered
# in Programs and Features, then stop the lingering process to release the lock.
#
# The installer defaults to PrivilegesRequired=admin, so it installs machine-wide
# when run elevated. "/ALLUSERS" is intentionally omitted: darktable's installer
# sets PrivilegesRequiredOverridesAllowed=dialog (not "commandline"), so the
# command-line override is not accepted and the admin default already covers all
# users.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 300
$pollIntervalSeconds = 5

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

# The old NSIS installer registered DisplayName "darktable"; the current Inno
# installer registers a versioned DisplayName (e.g. "darktable 5.6.0") with a
# darktable publisher. Match both.
function Test-DarktableInstalled {
    try {
        $props = Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object {
                $_.DisplayName -and
                ($_.DisplayName -eq 'darktable' -or $_.DisplayName -like 'darktable *') -and
                ($_.Publisher -like '*darktable*')
            } |
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
    Write-Host "Launched darktable installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        if (Test-DarktableInstalled) {
            Write-Host "darktable registered in Programs and Features after ${elapsed}s"
            if (-not $process.HasExited) {
                Stop-ProcessTree -ParentId $process.Id
                Write-Host "Stopped lingering installer process to release file lock"
            }
            Exit 0
        }

        # If Setup exits on its own, trust its exit code (after a final check).
        if ($process.HasExited) {
            Start-Sleep -Seconds 2
            if (Test-DarktableInstalled) { Exit 0 }
            $exitCode = $process.ExitCode
            Write-Host "Installer exited with code $exitCode but darktable was not detected"
            # 3010 = success, reboot required.
            if ($exitCode -eq 3010) { Exit 0 }
            Exit $exitCode
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (Test-DarktableInstalled) {
        if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for darktable to register in Programs and Features"
    if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
