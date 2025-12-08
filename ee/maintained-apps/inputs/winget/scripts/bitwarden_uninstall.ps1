# Attempts to locate Bitwarden's uninstaller from registry and execute it silently

$displayName = "Bitwarden"
$publisher = "8bit Solutions LLC"

# Some uninstallers require a flag to run silently.
# NSIS installers typically use "/S" for silent uninstall
$uninstallArgs = "/S"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

Write-Host "Starting Bitwarden uninstall script"
Write-Host "Searching for DisplayName: $displayName"
Write-Host "Searching for Publisher: $publisher"

try {
    # Kill any running Bitwarden processes before uninstalling
    Stop-Process -Name "Bitwarden" -Force -ErrorAction SilentlyContinue

    [array]$uninstallKeys = Get-ChildItem `
        -Path $paths `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath }

    Write-Host "Found $($uninstallKeys.Count) total uninstall entries in registry"
    
    # Debug: List all Bitwarden-like entries
    $bitwardenLike = $uninstallKeys | Where-Object { $_.DisplayName -and $_.DisplayName -like "*Bitwarden*" }
    Write-Host "Found $($bitwardenLike.Count) entries matching '*Bitwarden*'"
    foreach ($entry in $bitwardenLike) {
        Write-Host "  - DisplayName: $($entry.DisplayName), Publisher: $($entry.Publisher)"
    }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        # More lenient matching - check DisplayName first, then publisher
        $nameMatches = $key.DisplayName -and ($key.DisplayName -eq $displayName -or $key.DisplayName -like "$displayName*")
        $publisherMatches = $publisher -eq "" -or $key.Publisher -eq $publisher -or $key.Publisher -like "*$publisher*"
        
        if ($nameMatches) {
            Write-Host "Checking entry: DisplayName='$($key.DisplayName)', Publisher='$($key.Publisher)'"
            if ($publisherMatches -or $publisher -eq "") {
                $foundUninstaller = $true
                $registryKeyPath = $key.PSPath
                Write-Host "Found Bitwarden installation: $($key.DisplayName)"
                Write-Host "Registry key: $registryKeyPath"
                Write-Host "UninstallString: $($key.UninstallString)"
                Write-Host "QuietUninstallString: $($key.QuietUninstallString)"
            
            # Prefer QuietUninstallString if available - it's designed for silent uninstalls
            # If QuietUninstallString exists, use it directly without modification
            if ($key.QuietUninstallString) {
                Write-Host "Using QuietUninstallString for silent uninstall"
                $uninstallCommand = $key.QuietUninstallString
                
                # Execute QuietUninstallString directly via cmd.exe
                # This is more reliable than parsing and reconstructing
                Write-Host "Executing: cmd.exe /c `"$uninstallCommand`""
                $process = Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "`"$uninstallCommand`"" -PassThru -Wait -NoNewWindow
                $exitCode = $process.ExitCode
                Write-Host "Uninstall exit code: $exitCode"
            } else {
                # Fall back to UninstallString and add /S for silent
                Write-Host "Using UninstallString with /S switch"
                $uninstallCommand = $key.UninstallString
                
                # Parse the uninstall command to separate executable from arguments
                $splitArgs = $uninstallCommand.Split('"')
                if ($splitArgs.Length -gt 1) {
                    if ($splitArgs.Length -eq 3) {
                        $existingArgs = $splitArgs[2].Trim()
                        $uninstallArgs = if ($existingArgs) { "$existingArgs /S" } else { "/S" }
                    } elseif ($splitArgs.Length -gt 3) {
                        Throw `
                            "Uninstall command contains multiple quoted strings. " +
                                "Please update the uninstall script.`n" +
                                "Uninstall command: $uninstallCommand"
                    }
                    $uninstallCommand = $splitArgs[1]
                }
                Write-Host "Uninstall executable: $uninstallCommand"
                Write-Host "Uninstall args: $uninstallArgs"

                $processOptions = @{
                    FilePath = $uninstallCommand
                    PassThru = $true
                    Wait = $true
                }
                if ($uninstallArgs -ne '') {
                    $processOptions.ArgumentList = "$uninstallArgs"
                }

                # Start process and track exit code
                $process = Start-Process @processOptions
                $exitCode = $process.ExitCode
                Write-Host "Uninstall exit code: $exitCode"
            }
            
            # Wait for registry to update after uninstall completes
            # NSIS uninstallers may take a moment to clean up registry entries
            if ($exitCode -eq 0) {
                Write-Host "Waiting for registry cleanup..."
                Start-Sleep -Seconds 5
                
                # Verify uninstall worked by checking registry again
                $stillInstalled = $false
                [array]$verifyKeys = Get-ChildItem `
                    -Path $paths `
                    -ErrorAction SilentlyContinue |
                        ForEach-Object { Get-ItemProperty $_.PSPath }
                
                # Check if Bitwarden is still in registry
                $stillInstalled = $false
                $registryKeyToRemove = $null
                
                # Use the stored registry key path if available, otherwise search again
                if ($registryKeyPath -and (Test-Path $registryKeyPath -ErrorAction SilentlyContinue)) {
                    $stillInstalled = $true
                    $registryKeyToRemove = $registryKeyPath
                    Write-Host "WARNING: Bitwarden registry entry still exists at: $registryKeyToRemove"
                } else {
                    # Fallback: search again
                    foreach ($verifyKey in $verifyKeys) {
                        if ($verifyKey.DisplayName -and ($verifyKey.DisplayName -eq $displayName -or $verifyKey.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $verifyKey.Publisher -eq $publisher -or $verifyKey.Publisher -like "*$publisher*")) {
                            $stillInstalled = $true
                            $registryKeyToRemove = $verifyKey.PSPath
                            Write-Host "WARNING: Bitwarden still found in registry after uninstall: $($verifyKey.DisplayName)"
                            break
                        }
                    }
                }
                
                if ($stillInstalled -and $registryKeyToRemove) {
                    Write-Host "Attempting to manually remove registry entry: $registryKeyToRemove"
                    try {
                        Remove-Item -Path $registryKeyToRemove -Force -Recurse -ErrorAction Stop
                        Write-Host "Successfully removed registry entry manually"
                        Start-Sleep -Seconds 2
                        
                        # Verify removal
                        $finalCheck = Get-ItemProperty $registryKeyToRemove -ErrorAction SilentlyContinue
                        if (-not $finalCheck) {
                            Write-Host "Verified: Registry entry successfully removed"
                        } else {
                            Write-Host "WARNING: Registry entry still exists after manual removal attempt"
                        }
                    } catch {
                        Write-Host "Failed to manually remove registry entry: $_"
                    }
                } elseif (-not $stillInstalled) {
                    Write-Host "Verified: Bitwarden successfully removed from registry"
                }
            }
            
            # Exit the loop once the software is found and uninstalled.
            break
            } else {
                Write-Host "  Publisher mismatch: expected '$publisher', found '$($key.Publisher)'"
            }
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "ERROR: Uninstaller for '$displayName' not found in registry."
        Write-Host "Searched paths:"
        foreach ($path in $paths) {
            Write-Host "  - $path"
        }
        # Change exit code to 0 if you don't want to fail if uninstaller is not
        # found. This could happen if program was already uninstalled.
        $exitCode = 0
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

