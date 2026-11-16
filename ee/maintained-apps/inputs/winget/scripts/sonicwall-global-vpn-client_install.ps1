$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 900
$msiTimeoutSeconds = 600

function Stop-ProcessTree {
    param([int]$ParentId)
    Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree -ParentId $_.ProcessId }
    Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
}

# Returns the exit code, or $null if the process was stopped at the timeout.
function Invoke-WithTimeout {
    param([string]$FilePath, [string]$Arguments, [int]$TimeoutSeconds)
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $FilePath
    $psi.Arguments = $Arguments
    $psi.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::Start($psi)
    if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
        Stop-ProcessTree -ParentId $process.Id
        return $null
    }
    return $process.ExitCode
}

# 5.x registers under the "SonicWall Global VPN Client" key; 4.x was an MSI
# registered under its ProductCode.
function Get-GvcEntries {
    $paths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    foreach ($p in $paths) {
        Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -eq 'Global VPN Client' -and $_.Publisher -like '*SonicWall*'
        }
    }
}

try {
    # The installer rejects non-x64 hosts, so stop before a 4.x client is removed.
    $cpuArchitecture = (Get-CimInstance Win32_Processor -ErrorAction Stop | Select-Object -First 1).Architecture
    if ($cpuArchitecture -ne 9) {
        Write-Host "Global VPN Client 5.x requires an x64 host (Win32_Processor architecture $cpuArchitecture)."
        Exit 1
    }

    $vi = (Get-Item -LiteralPath $exeFilePath).VersionInfo
    $targetVersion = [version]::new($vi.ProductMajorPart, $vi.ProductMinorPart, $vi.ProductBuildPart, $vi.ProductPrivatePart)
    Write-Host "Installer version: $targetVersion"

    Stop-Process -Name 'SWGVC' -Force -ErrorAction SilentlyContinue

    # The 5.x installer can't replace 4.x: with /S it fails to register its
    # service and waits on an error dialog that never closes.
    $restartRequired = $false
    $legacy = @(Get-GvcEntries | Where-Object { $_.PSChildName -match '^\{[0-9A-Fa-f-]{36}\}$' })
    foreach ($entry in $legacy) {
        Write-Host "Removing Global VPN Client $($entry.DisplayVersion) ($($entry.PSChildName))"
        $msiExitCode = Invoke-WithTimeout -FilePath 'msiexec.exe' -Arguments "/x $($entry.PSChildName) /qn /norestart" -TimeoutSeconds $msiTimeoutSeconds
        if ($null -eq $msiExitCode) {
            Write-Host "Removal of Global VPN Client $($entry.DisplayVersion) did not finish within $msiTimeoutSeconds seconds. Stopped it."
            Exit 1
        }
        if (@(0, 1605, 1614, 1641, 3010) -notcontains $msiExitCode) {
            Write-Host "Failed to remove Global VPN Client $($entry.DisplayVersion): exit code $msiExitCode"
            Exit $msiExitCode
        }
        if (@(1641, 3010) -contains $msiExitCode) {
            $restartRequired = $true
        }
    }

    if ($restartRequired) {
        Write-Host "Removing the previous Global VPN Client requires a restart. Restart the host and try again."
        Exit 1
    }

    $current = @(Get-GvcEntries | Where-Object { $_.PSChildName -eq 'SonicWall Global VPN Client' })
    if ($current.Count -eq 0 -and (Get-Service -Name 'SWGVCSvc' -ErrorAction SilentlyContinue)) {
        Write-Host "The SWGVCSvc service from a previous Global VPN Client install is still registered. Restart the host and try again."
        Exit 1
    }

    # A failed service or driver step shows a dialog even with /S.
    $exitCode = Invoke-WithTimeout -FilePath $exeFilePath -Arguments '/S' -TimeoutSeconds $timeoutSeconds
    if ($null -eq $exitCode) {
        Write-Host "Installer did not finish within $timeoutSeconds seconds (likely waiting on an error dialog). Stopped it."
        Exit 1
    }

    Write-Host "Install exit code: $exitCode"
    if ($exitCode -ne 0) {
        Exit $exitCode
    }

    $installed = Get-GvcEntries | Where-Object { $_.PSChildName -eq 'SonicWall Global VPN Client' } | Select-Object -First 1
    if (-not $installed) {
        Write-Host "Installer exited 0 but Global VPN Client is not registered."
        Exit 1
    }

    $installedVersion = [string]$installed.DisplayVersion -as [version]
    if ($installedVersion) {
        $installedVersion = [version]::new($installedVersion.Major, $installedVersion.Minor, [Math]::Max($installedVersion.Build, 0), [Math]::Max($installedVersion.Revision, 0))
        if ($installedVersion -lt $targetVersion) {
            Write-Host "Installer exited 0 but Global VPN Client $($installed.DisplayVersion) is registered instead of $targetVersion."
            Exit 1
        }
    }

    Write-Host "Installed Global VPN Client $($installed.DisplayVersion)"
    Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
