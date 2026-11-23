# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# 3DxWare 10 is a self-extractor that runs a WiX Burn bundle. The self-extractor
# doesn't pass the bundle's exit code through, so the result is read from the
# Burn log and confirmed against the bundle's Programs and Features entry.
# The installer stops the 3DxWare driver (3DxService.exe) and starts it again
# when it's done, so this script must not wait on the installer's child processes.

$exeFilePath = "${env:INSTALLER_PATH}"
$displayName = "3Dconnexion 3DxWare 10"
$publisher = "3Dconnexion"
$logFile = Join-Path $env:TEMP "3DxWare10_install_$([guid]::NewGuid().ToString('N')).log"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-InstalledBundleVersion {
    $versions = @(Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object {
            $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher -and
            ($_.BundleUpgradeCode -or $_.QuietUninstallString)
        } |
        ForEach-Object {
            $v = $null
            if ([version]::TryParse("$($_.DisplayVersion)", [ref]$v)) { $v }
        })
    if ($versions.Count -eq 0) { return $null }
    return ($versions | Sort-Object -Descending | Select-Object -First 1)
}

function Get-BurnExitCode {
    if (-not (Test-Path $logFile)) { return $null }
    $line = Get-Content -Path $logFile -ErrorAction SilentlyContinue |
        Select-String -CaseSensitive -Pattern 'Exit code: 0x([0-9A-Fa-f]+), restarting' | Select-Object -Last 1
    if (-not $line) { return $null }
    return [Convert]::ToInt32($line.Matches[0].Groups[1].Value, 16)
}

function Write-LogTail {
    if (Test-Path $logFile) {
        Write-Host "Last lines of $($logFile):"
        Get-Content -Path $logFile -Tail 40 | ForEach-Object { Write-Host $_ }
    }
}

try {
    $targetVersion = $null
    [void][version]::TryParse("$((Get-Item $exeFilePath).VersionInfo.ProductVersion)".Trim(), [ref]$targetVersion)

    $process = Start-Process -FilePath $exeFilePath `
        -ArgumentList "/quiet /norestart /log `"$logFile`"" `
        -PassThru
    $null = $process.Handle

    $burnExitCode = $null
    $sw = [Diagnostics.Stopwatch]::StartNew()
    while (-not $process.HasExited) {
        $burnExitCode = Get-BurnExitCode
        if ($null -ne $burnExitCode) {
            [void]$process.WaitForExit(60000)
            break
        }
        if ($sw.Elapsed.TotalMinutes -ge 45) {
            Write-Host "The installer didn't finish within 45 minutes."
            Write-LogTail
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Exit 1
        }
        Start-Sleep -Seconds 5
    }

    $exitCode = 0
    if ($process.HasExited) {
        $exitCode = $process.ExitCode
        Write-Host "Self-extractor exit code: $exitCode"
    }
    if ($null -eq $burnExitCode) { $burnExitCode = Get-BurnExitCode }
    if ($null -ne $burnExitCode) {
        $exitCode = $burnExitCode
        Write-Host "Bundle exit code: $exitCode"
    }

    # 3010 = success, restart required; 1641 = success, restart initiated.
    if ($exitCode -notin @(0, 1641, 3010)) {
        Write-LogTail
        Exit $exitCode
    }

    $installedVersion = $null
    $sw = [Diagnostics.Stopwatch]::StartNew()
    while ($sw.Elapsed.TotalSeconds -lt 60) {
        $installedVersion = Get-InstalledBundleVersion
        if ($installedVersion -and (-not $targetVersion -or $installedVersion -ge $targetVersion)) { break }
        Start-Sleep -Seconds 2
    }

    if (-not $installedVersion -or ($targetVersion -and $installedVersion -lt $targetVersion)) {
        Write-Host "$displayName $targetVersion isn't registered after install (found: $installedVersion)."
        Write-LogTail
        Exit 1
    }

    Write-Host "$displayName $installedVersion is installed."
    Remove-Item -Path $logFile -Force -ErrorAction SilentlyContinue
    Exit 0

} catch {
    Write-Host "Error: $_"
    Write-LogTail
    Exit 1
}
