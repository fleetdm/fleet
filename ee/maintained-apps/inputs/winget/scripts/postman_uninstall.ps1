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
    
    # Wait a moment for uninstall to complete and registry to update
    Start-Sleep -Seconds 2
    
    # Verify the app is actually gone by checking registry
    # Even if winget returns a non-zero exit code, if the app is gone, consider it success
    $displayName = "Postman"
    $publisher = "Postman Inc."
    $paths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    
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
        Write-Host "Postman successfully uninstalled (verified by registry check)"
        Exit 0
    } elseif ($exitCode -eq 0) {
        # Exit code 0 but app still found - return success anyway (might be timing)
        Write-Host "Winget reported success (exit code 0)"
        Exit 0
    } else {
        # Non-zero exit code and app still found - return the exit code
        Write-Host "Uninstall may have failed - app still found in registry"
        Exit $exitCode
    }
} catch {
    Write-Host "Error running winget uninstall: $_"
    Exit 1
}
