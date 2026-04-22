# Microsoft Office (Click-to-Run) bootstrap installer.
# setup.exe is a small (~7 MB) Office Deployment Tool bootstrap; it downloads
# the configuration XML from the URL passed with /configure, then downloads
# and installs the full Office product set defined in that configuration.
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$configUrl = "https://aka.ms/fhlwingetconfig"

$exitCode = 0

try {
    # setup.exe /configure runs synchronously. It blocks until Click-to-Run
    # has downloaded and applied all selected products, which can take
    # 15-60+ minutes depending on network speed and selected products.
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/configure `"$configUrl`""
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    Write-Host "Starting Microsoft Office setup: $exeFilePath /configure $configUrl"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "setup.exe exit code: $exitCode"

    if ($exitCode -ne 0) {
        Exit $exitCode
    }

    # On some builds, the bootstrap returns before Click-to-Run has finished
    # writing the ARP uninstall entry. Poll briefly so that subsequent
    # osquery inventory picks the app up on the next run.
    $maxWaitSeconds = 600
    $elapsed = 0
    $registered = $null
    while ($elapsed -lt $maxWaitSeconds -and -not $registered) {
        foreach ($root in @(
                'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
                'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
            )) {
            $match = Get-ItemProperty "$root\*" -ErrorAction SilentlyContinue | Where-Object {
                $_.Publisher -eq 'Microsoft Corporation' -and
                $_.DisplayName -and
                ($_.DisplayName -like 'Microsoft 365*' -or $_.DisplayName -like 'Microsoft Office*') -and
                $_.UninstallString -like '*OfficeClickToRun.exe*'
            }
            if ($match) {
                $registered = $match | Select-Object -First 1
                break
            }
        }
        if (-not $registered) {
            Start-Sleep -Seconds 10
            $elapsed += 10
        }
    }

    if ($registered) {
        Write-Host "Detected installed product: $($registered.DisplayName) ($($registered.DisplayVersion))"
    } else {
        Write-Host "Warning: Microsoft Office uninstall entry not detected after $maxWaitSeconds seconds"
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
