$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 3010, 1641)

try {
    $installDir = Join-Path $env:ProgramFiles "BurpSuiteCommunity"

    # Setup exits with code 1 while Burp Suite is running. Fleet runs as SYSTEM,
    # outside the user's session, so it can't ask the app to close: stop it
    # first, along with anything else running from its install directory, such
    # as Burp's browser or the bundled Java runtime. Unsaved changes are lost.
    $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $_.Path -and $_.Path.StartsWith("$installDir\", [StringComparison]::OrdinalIgnoreCase)
    })
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

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
