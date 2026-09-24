# Agent Ransack ships as a zip containing the x64 MSI (agentransack_x64_<build>.msi).
# Fleet downloads the zip to INSTALLER_PATH; this script extracts it and installs
# the MSI per-machine and silently. The MSI sets ALLUSERS=1, so it always
# installs machine-wide.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "AgentRansackInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $msi = Get-ChildItem -Path $extractPath -Filter "*.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $msi) {
        Write-Host "Error: MSI not found under $extractPath"
        Exit 1
    }

    $logFile = Join-Path $env:TEMP "AgentRansackInstall.log"
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
