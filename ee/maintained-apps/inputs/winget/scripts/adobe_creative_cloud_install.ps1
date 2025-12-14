# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {
    Write-Host "Starting Adobe Creative Cloud installation (EXE installer with --mode=stub flag)..."
    
    # Adobe Creative Cloud uses --mode=stub for silent installation
    $processOptions = @{
        FilePath = $exeFilePath
        ArgumentList = "--mode=stub"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    
    Write-Host "Install exit code: $exitCode"
    
    # Adobe Creative Cloud installer may return exit code 1 even on successful installation
    # Verify installation by checking for the executable file
    if ($exitCode -eq 1) {
        $creativeCloudExe = Join-Path $env:ProgramFiles "Adobe\Adobe Creative Cloud\ACC\Creative Cloud.exe"
        if (Test-Path $creativeCloudExe) {
            Write-Host "Adobe Creative Cloud executable found at $creativeCloudExe. Treating exit code 1 as success."
            $exitCode = 0
        }
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

