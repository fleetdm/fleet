$exeFilePath = "${env:INSTALLER_PATH}"

# Define an array of common silent install parameters to try
$silentParams = @("/S", "/s", "/silent", "/quiet", "-s", "--silent", "/SILENT", "/VERYSILENT")

$installSuccess = $false
$finalExitCode = 1  # Default to failure

try {
    foreach ($param in $silentParams) {
        Write-Host "Attempting installation with parameter: $param"
        
        $processOptions = @{
            FilePath = "$exeFilePath"
            ArgumentList = "$param"
            PassThru = $true
            Wait = $true
        }
        
        # Start process and track exit code
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        
        Write-Host "Install exit code: $exitCode"
        
        # Check if installation was successful (typically exit code 0)
        if ($exitCode -eq 0) {
            Write-Host "Installation successful with parameter: $param"
            $installSuccess = $true
            $finalExitCode = 0
            break  # Exit the loop if installation was successful
        }
        
        Write-Host "Installation with parameter $param failed. Trying next parameter..."
    }
    
    if (-not $installSuccess) {
        Write-Host "All installation attempts failed."
    }
    
    Exit $finalExitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
