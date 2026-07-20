# RealVNC Viewer ships as a zip containing both the 32-bit and 64-bit MSIs
# (VNC-Viewer-<ver>-Windows-en-64bit.msi). Fleet downloads the zip to INSTALLER_PATH;
# this script extracts it and installs the 64-bit MSI per-machine and silently.
# The MSI sets ALLUSERS=1, so it always installs machine-wide.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "RealVNCViewerInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    # Prefer the 64-bit MSI; fall back to any *64bit*.msi found.
    $msi = Get-ChildItem -Path $extractPath -Filter "*64bit*.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $msi) {
        Write-Host "Error: 64-bit MSI not found under $extractPath"
        Exit 1
    }

    $logFile = Join-Path $env:TEMP "RealVNCViewerInstall.log"
    $process = Start-Process -FilePath "msiexec.exe" `
        -ArgumentList "/i `"$($msi.FullName)`" /quiet /norestart /l*v `"$logFile`"" `
        -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code (msiexec): $exitCode"

    Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

    # 3010 = success, reboot required; 1641 = success, reboot initiated.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
