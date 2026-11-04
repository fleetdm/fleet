# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Citrix Workspace app ships a vendor bootstrapper (CitrixWorkspaceApp.exe).
# /silent installs machine-wide without UI; /noreboot keeps it from rebooting the
# host; /AutoUpdateCheck=disabled leaves updates to Fleet rather than the app.
#
# The bootstrapper installs its component MSIs and then does not exit on its own
# (it leaves the Workspace UI / connection-center processes running), so a plain
# "Start-Process -Wait" blocks until the install timeout even though the app
# installed correctly. Instead we launch it, poll until "Citrix Workspace" is
# registered in Programs and Features, then stop the lingering bootstrapper tree.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 540
$pollIntervalSeconds = 10

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

# Matches the shipped exists query: a versioned/inconsistent DisplayName and a
# publisher that recent builds moved from "Citrix Systems, Inc." to "Cloud
# Software Group". "Citrix Workspace Inside" is a bundled sub-component with its
# own (lower) version that registers early; excluding it keeps the poll from
# killing the bootstrapper before the main app is registered.
function Test-CitrixInstalled {
    try {
        $props = Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object {
                $_.DisplayName -like 'Citrix Workspace*' -and
                $_.DisplayName -notlike '*Inside*' -and
                ($_.Publisher -like 'Citrix*' -or $_.Publisher -like 'Cloud Software Group*')
            } |
            Select-Object -First 1
        return [bool]$props
    } catch {
        return $false
    }
}

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
        ArgumentList = "/silent", "/noreboot", "/AutoUpdateCheck=disabled"
        PassThru = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Launched Citrix Workspace installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        if (Test-CitrixInstalled) {
            Write-Host "Citrix Workspace registered in Programs and Features after ${elapsed}s"
            if (-not $process.HasExited) {
                Stop-ProcessTree -ParentId $process.Id
                Write-Host "Stopped lingering bootstrapper process"
            }
            Exit 0
        }

        # If the bootstrapper exits on its own, trust its exit code (after a final check).
        if ($process.HasExited) {
            Start-Sleep -Seconds 3
            if (Test-CitrixInstalled) { Exit 0 }
            $exitCode = $process.ExitCode
            Write-Host "Installer exited with code $exitCode but Citrix Workspace was not detected"
            # 3010 = reboot required, 1641 = reboot initiated: both are successful installs.
            if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
            Exit $exitCode
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (Test-CitrixInstalled) {
        if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for Citrix Workspace to register"
    if (-not $process.HasExited) { Stop-ProcessTree -ParentId $process.Id }
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
