# Okta Verify ships as a WiX Burn bootstrapper. Silent switches come from the
# winget installer type convention for Burn bundles.

$exeFilePath = "${env:INSTALLER_PATH}"
$expectedExitCodes = @(0, 1641, 3010)

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/quiet /norestart"
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($expectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error: $_"
    Exit 1
}
