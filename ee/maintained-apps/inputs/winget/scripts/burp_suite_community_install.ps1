$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 3010, 1641)

try {
    $installDir = Join-Path $env:ProgramFiles "BurpSuiteCommunity"

    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "-q", "-Dinstall4j.suppressUnattendedReboot=true", "-dir", $installDir
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    Start-Sleep -Seconds 5

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
