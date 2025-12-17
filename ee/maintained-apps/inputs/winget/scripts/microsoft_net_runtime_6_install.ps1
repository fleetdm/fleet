# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {
    Write-Host "Starting Microsoft .NET Runtime 6 installation..."
    
    # Microsoft .NET Runtime uses a burn installer (WiX bootstrapper)
    # Per Winget manifest: Silent: /quiet, Custom: /norestart
    $processOptions = @{
        FilePath = $exeFilePath
        ArgumentList = "/quiet", "/norestart"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    
    Write-Host "Install exit code: $exitCode"
    
    # Burn installers may return non-zero exit codes even on success
    # Verify installation by checking registry if exit code is non-zero
    if ($exitCode -ne 0) {
        Write-Host "Installer exited with non-zero code: $exitCode. Verifying installation..."
        Start-Sleep -Seconds 3
        
        $registryPaths = @(
            "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
            "HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
        )
        
        $found = $false
        foreach ($path in $registryPaths) {
            $entry = Get-ItemProperty $path -ErrorAction SilentlyContinue | Where-Object {
                $_.DisplayName -like "*Microsoft .NET Runtime*6.0*" -and
                $_.Publisher -eq "Microsoft Corporation"
            }
            if ($entry) {
                Write-Host "Microsoft .NET Runtime 6 found in registry: $($entry.DisplayName) (Version: $($entry.DisplayVersion))"
                $found = $true
                $exitCode = 0 # Override exit code to 0 for successful validation
                break
            }
        }
        
        if (-not $found) {
            Write-Host "Microsoft .NET Runtime 6 not found in registry. Installation may have failed."
        }
    } else {
        # Even if exit code is 0, give osquery a moment to detect the installation
        Start-Sleep -Seconds 2
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

