$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 900

function Stop-ProcessTree {
    param([int]$ParentId)
    Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree -ParentId $_.ProcessId }
    Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
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
    Stop-Process -Name 'SWGVC' -Force -ErrorAction SilentlyContinue

    # The 5.x installer can't replace 4.x: with /S it fails to register its
    # service and waits on an error dialog that never closes.
    $legacy = @(Get-GvcEntries | Where-Object { $_.PSChildName -match '^\{[0-9A-Fa-f-]{36}\}$' })
    foreach ($entry in $legacy) {
        Write-Host "Removing Global VPN Client $($entry.DisplayVersion) ($($entry.PSChildName))"
        $msi = Start-Process -FilePath 'msiexec.exe' -ArgumentList "/x $($entry.PSChildName) /qn /norestart" -Wait -PassThru
        if (@(0, 1605, 1614, 1641, 3010) -notcontains $msi.ExitCode) {
            Write-Host "Failed to remove Global VPN Client $($entry.DisplayVersion): exit code $($msi.ExitCode)"
            Exit $msi.ExitCode
        }
    }

    $current = @(Get-GvcEntries | Where-Object { $_.PSChildName -eq 'SonicWall Global VPN Client' })
    if ($current.Count -eq 0 -and (Get-Service -Name 'SWGVCSvc' -ErrorAction SilentlyContinue)) {
        Write-Host "The SWGVCSvc service from a previous Global VPN Client install is still registered. Restart the host and try again."
        Exit 1
    }

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $exeFilePath
    $psi.Arguments = '/S'
    $psi.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::Start($psi)

    # A failed service or driver step shows a dialog even with /S.
    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Write-Host "Installer did not finish within $timeoutSeconds seconds (likely waiting on an error dialog). Stopping it."
        Stop-ProcessTree -ParentId $process.Id
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if ($exitCode -ne 0) {
        Exit $exitCode
    }

    $installed = Get-GvcEntries | Where-Object { $_.PSChildName -eq 'SonicWall Global VPN Client' } | Select-Object -First 1
    if (-not $installed) {
        Write-Host "Installer exited 0 but Global VPN Client is not registered."
        Exit 1
    }

    Write-Host "Installed Global VPN Client $($installed.DisplayVersion)"
    Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
