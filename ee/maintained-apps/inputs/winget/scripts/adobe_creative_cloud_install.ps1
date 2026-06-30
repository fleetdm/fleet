# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Runs the Adobe Creative Cloud stub with silent flags, waits until the app is
# registered in Programs and Features with a matching version, then stops the
# stub process so the installer file can be removed from disk.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 240
$pollIntervalSeconds = 10

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Get-ExpectedVersionFromStub {
    $vi = [System.Diagnostics.FileVersionInfo]::GetVersionInfo($exeFilePath)
    $v = $vi.ProductVersion
    if ([string]::IsNullOrWhiteSpace($v)) { $v = $vi.FileVersion }
    if ([string]::IsNullOrWhiteSpace($v)) { return $null }
    $v = $v.Trim()
    if ($v -match '^([\d.]+)') { return $Matches[1] }
    return ($v -split '\s+')[0]
}

function Get-CreativeCloudUninstallProps {
    try {
        return Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object {
                $_.DisplayName -eq 'Adobe Creative Cloud' -and
                $_.Publisher -eq 'Adobe Inc.'
            } |
            Select-Object -First 1
    } catch {
        return $null
    }
}

function Test-VersionCompatible {
    param(
        [string]$Found,
        [string]$Expected
    )
    if ([string]::IsNullOrWhiteSpace($Found) -or [string]::IsNullOrWhiteSpace($Expected)) {
        return $false
    }
    if ($Found -eq $Expected) { return $true }
    if ($Found.StartsWith($Expected + '.')) { return $true }
    if ($Expected.StartsWith($Found + '.')) { return $true }
    return $false
}

function Test-CreativeCloudReady {
    param([string]$ExpectedVersion)

    $props = Get-CreativeCloudUninstallProps
    if (-not $props) { return $false }

    $displayVersion = [string]$props.DisplayVersion
    if ([string]::IsNullOrWhiteSpace($displayVersion)) { return $false }

    return (Test-VersionCompatible -Found $displayVersion.Trim() -Expected $ExpectedVersion)
}

try {
    $expectedVersion = Get-ExpectedVersionFromStub
    if (-not $expectedVersion) {
        Write-Host "Could not read version from stub installer"
        Exit 1
    }
    Write-Host "Expecting inventory version compatible with: $expectedVersion"

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "--mode=stub"
        PassThru = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Launched Creative Cloud stub installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        if (Test-CreativeCloudReady -ExpectedVersion $expectedVersion) {
            Write-Host "Adobe Creative Cloud registered with matching version after ${elapsed}s"
            if (-not $process.HasExited) {
                Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
                Write-Host "Stopped stub installer process to release installer file lock"
            }
            Exit 0
        }

        if ($process.HasExited) {
            $exitCode = $process.ExitCode
            Write-Host "Stub installer exited with code $exitCode after ${elapsed}s"
            if (Test-CreativeCloudReady -ExpectedVersion $expectedVersion) {
                Exit 0
            }
            Exit $exitCode
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (Test-CreativeCloudReady -ExpectedVersion $expectedVersion) {
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for Adobe Creative Cloud to appear in Programs and Features with a matching version"
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
