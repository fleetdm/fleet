$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 3010, 1641)

try {
    # Without -dir, install4j defaults to a per-user install even when
    # elevated, so the Program Files override is load-bearing.
    $installDir = Join-Path $env:ProgramFiles "BurpSuitePro"

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
