# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# GOG Galaxy ships as an Inno Setup installer.

$exeFilePath = "${env:INSTALLER_PATH}"
$logFilePath = Join-Path $env:TEMP "gog-galaxy-install.log"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS /LOG=`"$logFilePath`""
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # On failure, surface the Inno Setup log so the error is visible in
    # Fleet's script output instead of just a bare exit code.
    if ($exitCode -ne 0 -and (Test-Path $logFilePath)) {
        Write-Host "--- Inno Setup log (last 50 lines) ---"
        Get-Content $logFilePath -Tail 50 | ForEach-Object { Write-Host $_ }
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
