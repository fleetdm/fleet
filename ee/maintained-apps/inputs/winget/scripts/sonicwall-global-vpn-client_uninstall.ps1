$timeoutSeconds = 600

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

function Invoke-Uninstaller {
    param([string]$FilePath, [string]$Arguments)
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $FilePath
    $psi.Arguments = $Arguments
    $psi.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::Start($psi)
    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Write-Host "Uninstaller did not finish within $timeoutSeconds seconds. Stopping it."
        Stop-ProcessTree -ParentId $process.Id
        return 1
    }
    return $process.ExitCode
}

try {
    $entries = @(Get-GvcEntries)
    if ($entries.Count -eq 0) {
        Write-Host "Global VPN Client is not installed"
        Exit 0
    }

    Stop-Process -Name 'SWGVC' -Force -ErrorAction SilentlyContinue

    foreach ($entry in $entries) {
        if ($entry.PSChildName -match '^\{[0-9A-Fa-f-]{36}\}$') {
            Write-Host "Uninstalling Global VPN Client $($entry.DisplayVersion) ($($entry.PSChildName))"
            $exitCode = Invoke-Uninstaller -FilePath 'msiexec.exe' -Arguments "/x $($entry.PSChildName) /qn /norestart"
            if (@(0, 1605, 1614, 1641, 3010) -notcontains $exitCode) {
                Write-Host "Uninstall exit code: $exitCode"
                Exit $exitCode
            }
            continue
        }

        $uninstallCommand = $entry.UninstallString
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exePath = $matches[1]
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $exePath = $matches[1]
        } else {
            Write-Host "Could not parse uninstall string: $uninstallCommand"
            Exit 1
        }

        if (-not (Test-Path -LiteralPath $exePath)) {
            Write-Host "Uninstaller not found: $exePath"
            Exit 1
        }

        # _?= runs the NSIS uninstaller in place so it can be waited on; it must be the last argument.
        $installDir = Split-Path -Path $exePath -Parent
        Write-Host "Uninstalling Global VPN Client $($entry.DisplayVersion) with $exePath"
        $exitCode = Invoke-Uninstaller -FilePath $exePath -Arguments "/S _?=$installDir"
        Write-Host "Uninstall exit code: $exitCode"
        if ($exitCode -ne 0) {
            Exit $exitCode
        }

        # Running in place leaves the uninstaller itself behind.
        Remove-Item -LiteralPath $exePath -Force -ErrorAction SilentlyContinue
        if ((Test-Path -LiteralPath $installDir) -and -not (Get-ChildItem -LiteralPath $installDir -Force -ErrorAction SilentlyContinue)) {
            Remove-Item -LiteralPath $installDir -Force -ErrorAction SilentlyContinue
        }
    }

    $remaining = @(Get-GvcEntries)
    if ($remaining.Count -gt 0) {
        Write-Host "Global VPN Client is still registered after uninstall."
        Exit 1
    }

    Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
