# DisplayLink ships one zip holding both the x64 and arm64 MSIs, so this script
# picks DisplayLink_Win10RS.msi (x64) by name rather than taking the first MSI
# it finds. Installing the graphics driver commonly asks for a reboot.

$zipFilePath = "${env:INSTALLER_PATH}"
$msiName = "DisplayLink_Win10RS.msi"

try {
    $extractPath = Join-Path $env:TEMP "DisplayLinkInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $msi = Get-ChildItem -Path $extractPath -Filter $msiName -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $msi) {
        Write-Host "Error: $msiName not found under $extractPath"
        Exit 1
    }

    $logFile = Join-Path $env:TEMP "DisplayLinkInstall.log"
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
