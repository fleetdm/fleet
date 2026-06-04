# VNC Viewer (RealVNC): winget ships this as a .zip containing both the 32-bit and
# 64-bit WiX (MSI) installers. Extract the archive, select the 64-bit MSI, and install
# it silently. The MSI sets ALLUSERS=1, so it always installs per-machine, which is
# what Fleet needs for its SYSTEM-context installs.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "VNCViewerInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    # Prefer the 64-bit MSI; fall back to any MSI if the naming changes.
    $msi = Get-ChildItem -Path $extractPath -Filter "*64bit*.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $msi) {
        $msi = Get-ChildItem -Path $extractPath -Filter "*.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    }

    if (-not $msi) {
        Write-Host "Error: no MSI found under $extractPath"
        Exit 1
    }

    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList "/i `"$($msi.FullName)`" /quiet /norestart" -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

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
