# CutePDF Writer ships as a zip (CuteWriter.zip) containing the Inno Setup
# installer (CuteWriter.exe) alongside the GPL Ghostscript PS2PDF converter
# (converter.exe). Fleet downloads the zip to INSTALLER_PATH; this script
# extracts the FULL zip and runs CuteWriter.exe from the extracted directory
# so the setup can find and install the adjacent converter (without it the
# printer produces no PDFs).

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "CutePDFWriterInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $setup = Get-ChildItem -Path $extractPath -Filter "CuteWriter.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $setup) {
        Write-Host "Error: CuteWriter.exe not found under $extractPath"
        Exit 1
    }

    $process = Start-Process -FilePath $setup.FullName `
        -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
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
