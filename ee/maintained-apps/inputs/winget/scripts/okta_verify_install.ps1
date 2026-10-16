# Okta Verify ships as a WiX Burn bootstrapper. Its setup app (7.0+) rejects
# any argument that isn't key=value except /quiet, /q, and /log, so passing
# /norestart makes it exit 1.

$exeFilePath = "${env:INSTALLER_PATH}"
$logPath = Join-Path $env:TEMP "OktaVerifyInstall.log"
$expectedExitCodes = @(0, 1641, 3010)

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/quiet /log `"$logPath`""
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($expectedExitCodes -contains $exitCode) { Exit 0 }
    if (Test-Path $logPath) {
        Write-Host "Last lines of the installer log ($logPath):"
        Get-Content $logPath -Tail 80 | ForEach-Object { Write-Host $_ }
    }
    Exit $exitCode
} catch {
    Write-Host "Error: $_"
    Exit 1
}
