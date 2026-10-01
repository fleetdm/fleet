# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Connect Fonts for Windows (Monotype Connect) ships as a zip holding a WiX Burn
# bundle. This script extracts the zip and runs the bundle silently; the bundle
# also installs the WebView2 runtime and VC++ redistributables when missing.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "MonotypeConnectInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $bundle = Get-ChildItem -Path $extractPath -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $bundle) {
        Write-Host "Error: installer .exe not found under $extractPath"
        Exit 1
    }

    $logFile = Join-Path $env:TEMP "MonotypeConnectInstall.log"
    $process = Start-Process -FilePath $bundle.FullName `
        -ArgumentList "/quiet /norestart /log `"$logFile`"" `
        -PassThru -Wait
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
