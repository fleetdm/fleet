# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Adobe Creative Cloud is distributed as a small "stub" executable. Running it
# with --mode=stub lays down the Adobe Creative Cloud Desktop app silently, and
# then continues downloading/installing the full suite in the background. The
# background work can run for tens of minutes, so we cannot simply -Wait on the
# installer process - the orchestrating script has its own timeout.
#
# Strategy: launch the stub, then poll for the Creative Cloud Desktop install
# artifacts (uninstaller exe and/or Uninstall registry entry). As soon as the
# app is registered we consider the install successful and exit 0.

$exeFilePath = "${env:INSTALLER_PATH}"

$pollTimeoutSeconds = 240
$pollIntervalSeconds = 10

$expectedUninstallerPaths = @(
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe",
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe"
)

$registryUninstallPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Test-CreativeCloudInstalled {
    foreach ($p in $expectedUninstallerPaths) {
        if (Test-Path -LiteralPath $p) { return $true }
    }

    try {
        $found = Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -eq 'Adobe Creative Cloud' -and $_.Publisher -eq 'Adobe Inc.' } |
            Select-Object -First 1
        if ($found) { return $true }
    } catch {}

    return $false
}

try {
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "--mode=stub"
        PassThru = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Launched Creative Cloud stub installer (PID: $($process.Id))"

    $elapsed = 0
    while ($elapsed -lt $pollTimeoutSeconds) {
        if (Test-CreativeCloudInstalled) {
            Write-Host "Adobe Creative Cloud detected after ${elapsed}s"
            Exit 0
        }

        if ($process.HasExited) {
            $exitCode = $process.ExitCode
            Write-Host "Stub installer exited with code $exitCode after ${elapsed}s"
            if (Test-CreativeCloudInstalled) {
                Exit 0
            }
            Exit $exitCode
        }

        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (Test-CreativeCloudInstalled) {
        Write-Host "Adobe Creative Cloud detected after polling timeout"
        Exit 0
    }

    Write-Host "Timed out after ${pollTimeoutSeconds}s waiting for Adobe Creative Cloud to register"
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
