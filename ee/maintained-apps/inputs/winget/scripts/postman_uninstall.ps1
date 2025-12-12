# Uninstall Postman using winget
# Since Postman is installed via winget, we can use winget to uninstall it directly

$packageIdentifier = "Postman.Postman"

try {
    Write-Host "Uninstalling Postman using winget..."
    
    $processOptions = @{
        FilePath = "winget"
        ArgumentList = @("uninstall", "--id", $packageIdentifier, "--silent")
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    
    Write-Host "Uninstall exit code: $exitCode"
    
    # Winget returns 0 on success
    # Non-zero exit codes can indicate various states (package not found, already uninstalled, etc.)
    # For validation purposes, we'll return the exit code as-is
    # The validator will check if the app is actually gone
    Exit $exitCode
} catch {
    Write-Host "Error running winget uninstall: $_"
    Exit 1
}
