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

try {
    # Kill any running Bitwarden processes before uninstalling
    Stop-Process -Name "Bitwarden" -Force -ErrorAction SilentlyContinue

    [array]$uninstallKeys = Get-ChildItem `
        -Path $paths `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -and ($key.DisplayName -eq $displayName -or $key.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $key.Publisher -eq $publisher)) {
            $foundUninstaller = $true
            Write-Host "Found Bitwarden installation: $($key.DisplayName)"
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
                
                foreach ($verifyKey in $verifyKeys) {
                    if ($verifyKey.DisplayName -and ($verifyKey.DisplayName -eq $displayName -or $verifyKey.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $verifyKey.Publisher -eq $publisher)) {
                        $stillInstalled = $true
                        Write-Host "WARNING: Bitwarden still found in registry after uninstall: $($verifyKey.DisplayName)"
                        break
                    }
                }
                
                if (-not $stillInstalled) {
                    Write-Host "Verified: Bitwarden successfully removed from registry"
                }
            }
            
            # Exit the loop once the software is found and uninstalled.
            break
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$displayName' not found."
        # Change exit code to 0 if you don't want to fail if uninstaller is not
        # found. This could happen if program was already uninstalled.
        $exitCode = 0
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

