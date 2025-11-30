# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Verify installer file exists
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing Sublime Text from: $exeFilePath"
    
    # Add arguments to install silently
    # Sublime Text uses an Inno Setup-based installer
    # Based on winget manifest: https://github.com/microsoft/winget-pkgs/blob/master/manifests/s/SublimeHQ/SublimeText/4/4.0.0.420000/SublimeHQ.SublimeText.4.installer.yaml
    # /VERYSILENT = Very silent installation (no dialogs, no progress bar)
    # /SUPPRESSMSGBOXES = Suppress message boxes
    # /NORESTART = Do not restart the computer
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    # Start process and track exit code
    Write-Host "Starting installation with arguments: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    
    if ($null -eq $process) {
        Write-Host "Error: Failed to start installer process"
        Exit 1
    }
    
    $exitCode = $process.ExitCode
    
    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    
    # Sublime Text's installer intentionally doesn't write version to registry
    # We need to manually write it so osquery can detect it
    if ($exitCode -eq 0) {
        Write-Host "Writing version to registry for osquery detection..."
        
        # Wait a moment for installation to complete and registry to be created
        Start-Sleep -Seconds 2
        
        # Get version from the installed executable's file version
        $sublimeExe = "C:\Program Files\Sublime Text\sublime_text.exe"
        $version = $null
        
        if (Test-Path $sublimeExe) {
            try {
                $fileVersionInfo = (Get-Item $sublimeExe).VersionInfo
                # Try FileVersion first, fall back to ProductVersion
                $version = if ($fileVersionInfo.FileVersion) { $fileVersionInfo.FileVersion } else { $fileVersionInfo.ProductVersion }
                Write-Host "Found version from executable: $version"
            } catch {
                Write-Host "Warning: Could not read version from executable: $_"
            }
        }
        
        # If we still don't have a version, try to extract from installer filename
        if (-not $version) {
            $installerName = Split-Path -Leaf $exeFilePath
            # Try to match build number pattern (e.g., "sublime_text_build_4200_x64_setup.exe")
            if ($installerName -match 'build_(\d+)') {
                $buildNumber = $matches[1]
                # Convert build number to version format (e.g., 4200 -> 4.0.0.420000)
                # This is a heuristic - actual version format may vary
                $major = if ($buildNumber.Length -ge 1) { $buildNumber[0] } else { "4" }
                $minor = "0"
                $patch = "0"
                $build = $buildNumber.PadRight(6, '0')
                $version = "$major.$minor.$patch.$build"
                Write-Host "Extracted version from filename: $version"
            }
        }
        
        # Write version to registry if we found it
        if ($version) {
            $registryPath = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Sublime Text_is1"
            if (Test-Path $registryPath) {
                try {
                    Set-ItemProperty -Path $registryPath -Name "DisplayVersion" -Value $version -ErrorAction Stop
                    Write-Host "Successfully wrote version '$version' to registry at DisplayVersion"
                } catch {
                    Write-Host "Warning: Failed to write DisplayVersion to registry: $_"
                }
            } else {
                Write-Host "Warning: Registry key not found at $registryPath"
            }
        } else {
            Write-Host "Warning: Could not determine version to write to registry"
        }
    }
    
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    Exit 1
}

