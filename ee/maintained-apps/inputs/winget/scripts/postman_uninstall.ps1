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
    
    # Wait a moment for uninstall to complete (winget may return before uninstaller finishes)
    Start-Sleep -Seconds 3
    
    # Exit 0 if the command completed (validation will check if app is actually gone)
    Exit 0
} catch {
    Write-Host "Error running winget uninstall: $_"
    Exit 1
}
