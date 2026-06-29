# Canva Desktop is an NSIS installer. NSIS uses /S for a silent install.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/S"
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # electron-builder NSIS often auto-launches the app after a silent install.
    # Stop it so files aren't left locked for later detection/uninstall.
    Start-Sleep -Seconds 5
    Stop-Process -Name "Canva" -Force -ErrorAction SilentlyContinue

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
