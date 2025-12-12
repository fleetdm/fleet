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
    
    # Wait for uninstaller to complete and registry to update
    # Winget may return before the actual uninstaller process finishes
    # Loop until Postman is removed from registry or timeout
    $maxWaitSeconds = 30
    $waitIntervalSeconds = 1
    $waitedSeconds = 0
    $displayName = "Postman"
    $publisher = "Postman Inc."
    $paths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    
    while ($waitedSeconds -lt $maxWaitSeconds) {
        Start-Sleep -Seconds $waitIntervalSeconds
        $waitedSeconds += $waitIntervalSeconds
        
        $stillInstalled = $false
        foreach ($p in $paths) {
            $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
                $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
            }
            if ($items) {
                $stillInstalled = $true
                break
            }
        }
        
        if (-not $stillInstalled) {
            Write-Host "Postman successfully removed from registry after $waitedSeconds seconds"
            Exit 0
        }
    }
    
    Write-Host "Timeout waiting for Postman to be removed from registry"
    Exit 0

} catch {
    Write-Host "Error running winget uninstall: $_"
    Exit 1
}
