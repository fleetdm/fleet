$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # NSIS installers support silent installation with the /S flag
    $processOptions = @{
        FilePath     = $exeFilePath
        ArgumentList = @("/S")
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Install exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error: $_"
    Exit 1
}
