# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = "Zoom Workplace"

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Define an array of common silent uninstall parameters to try
$silentParams = @("/S", "/s", "/silent", "/quiet", "-s", "--silent", "/SILENT", "/VERYSILENT", "/NORESTART", "-q", "--quiet", "/uninstall")

$machineKey = `
'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {
    [array]$uninstallKeys = Get-ChildItem `
        -Path @($machineKey, $machineKey32on64) `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        # If needed, add -notlike to the comparison to exclude certain similar
        # software
        if ($key.DisplayName -like $softwareNameLike) {
            $foundUninstaller = $true
            Write-Host "Found software: $($key.DisplayName)"
            
            # Get the uninstall string without any arguments
            $baseUninstallCommand = $key.UninstallString
            
            # The uninstall command may contain command and args, like:
            # "C:\Program Files\Software\uninstall.exe" --uninstall --silent
            # Extract just the executable path
            $splitArgs = $baseUninstallCommand.Split('"')
            if ($splitArgs.Length -gt 1) {
                $baseUninstallCommand = $splitArgs[1]
                # Get any existing arguments
                $existingArgs = ""
                if ($splitArgs.Length -ge 3) {
                    $existingArgs = $splitArgs[2].Trim()
                }
            }
            
            Write-Host "Base uninstall command: $baseUninstallCommand"
            
            $uninstallSuccess = $false
            
            # First, try QuietUninstallString if it exists
            if ($key.QuietUninstallString) {
                Write-Host "Trying QuietUninstallString: $($key.QuietUninstallString)"
                
                $processOptions = @{
                    FilePath = $baseUninstallCommand
                    PassThru = $true
                    Wait = $true
                }
                
                # Extract arguments from QuietUninstallString if they exist
                $quietSplitArgs = $key.QuietUninstallString.Split('"')
                if ($quietSplitArgs.Length -ge 3) {
                    $quietArgs = $quietSplitArgs[2].Trim()
                    if ($quietArgs) {
                        $processOptions.ArgumentList = "$quietArgs"
                    }
                }
                
                # Start process and track exit code
                $process = Start-Process @processOptions
                $exitCode = $process.ExitCode
                
                Write-Host "QuietUninstallString exit code: $exitCode"
                
                if ($exitCode -eq 0) {
                    Write-Host "Uninstallation successful with QuietUninstallString"
                    $uninstallSuccess = $true
                }
            }
            
            # If QuietUninstallString didn't work or doesn't exist, try each silent parameter
            if (-not $uninstallSuccess) {
                foreach ($param in $silentParams) {
                    Write-Host "Attempting uninstallation with parameter: $param"
                    
                    # Combine existing args with silent parameter
                    $combinedArgs = if ($existingArgs) {
                        "$existingArgs $param"
                    } else {
                        "$param"
                    }
                    
                    $processOptions = @{
                        FilePath = "$baseUninstallCommand"
                        ArgumentList = "$combinedArgs"
                        PassThru = $true
                        Wait = $true
                    }
                    
                    # Start process and track exit code
                    $process = Start-Process @processOptions
                    $exitCode = $process.ExitCode
                    
                    Write-Host "Uninstall exit code: $exitCode"
                    
                    # Check if uninstallation was successful (typically exit code 0)
                    if ($exitCode -eq 0) {
                        Write-Host "Uninstallation successful with parameter: $param"
                        $uninstallSuccess = $true
                        break  # Exit the loop if uninstallation was successful
                    }
                    
                    Write-Host "Uninstallation with parameter $param failed. Trying next parameter..."
                    
                    # Add a short delay between attempts
                    Start-Sleep -Seconds 2
                }
            }
            
            # If none of the parameters worked, try specifically for Zoom which might use special arguments
            if (-not $uninstallSuccess) {
                Write-Host "Trying Zoom-specific uninstall parameters"
                
                # Zoom often uses /uninstall /silent as its parameters
                $zoomParams = @("/uninstall /silent", "--uninstall --silent", "/uninstall /quiet")
                
                foreach ($param in $zoomParams) {
                    Write-Host "Attempting uninstallation with Zoom-specific parameter: $param"
                    
                    $processOptions = @{
                        FilePath = "$baseUninstallCommand"
                        ArgumentList = "$param"
                        PassThru = $true
                        Wait = $true
                    }
                    
                    # Start process and track exit code
                    $process = Start-Process @processOptions
                    $exitCode = $process.ExitCode
                    
                    Write-Host "Uninstall exit code: $exitCode"
                    
                    # Check if uninstallation was successful (typically exit code 0)
                    if ($exitCode -eq 0) {
                        Write-Host "Uninstallation successful with Zoom-specific parameter: $param"
                        $uninstallSuccess = $true
                        break  # Exit the loop if uninstallation was successful
                    }
                    
                    Write-Host "Uninstallation with Zoom-specific parameter $param failed. Trying next parameter..."
                    
                    # Add a short delay between attempts
                    Start-Sleep -Seconds 2
                }
            }
            
            if (-not $uninstallSuccess) {
                Write-Host "All uninstallation attempts failed."
                $exitCode = 1
            }
            
            # Exit the loop once the software is found and uninstallation is attempted
            break
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$softwareName' not found."
        # Change exit code to 0 if you don't want to fail if uninstaller is not
        # found. This could happen if program was already uninstalled.
        $exitCode = 1
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
