# Slack Uninstall Script
# This script handles both MSI and EXE installations, including per-user installations

$softwareName = "Slack"
$exitCode = 0
$uninstalled = $false

Write-Host "Starting Slack uninstallation process..."

# Function to uninstall MSI packages
function Remove-SlackMSI {
    Write-Host "Checking for MSI-based Slack installations..."
    
    # Find all Slack MSI products
    $msiProducts = Get-WmiObject -Class Win32_Product -Filter "Name LIKE '%Slack%'" -ErrorAction SilentlyContinue
    
    if ($msiProducts) {
        foreach ($product in $msiProducts) {
            Write-Host "Found MSI: $($product.Name) - Version: $($product.Version)"
            Write-Host "Attempting to uninstall MSI..."
            
            try {
                $result = $product.Uninstall()
                if ($result.ReturnValue -eq 0) {
                    Write-Host "Successfully uninstalled MSI: $($product.Name)"
                    return $true
                } else {
                    Write-Host "MSI uninstall returned code: $($result.ReturnValue)"
                }
            } catch {
                Write-Host "Error uninstalling MSI: $_"
            }
        }
    }
    
    # Also try using msiexec with product codes from registry
    $msiKeys = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )
    
    foreach ($keyPath in $msiKeys) {
        $keys = Get-ChildItem -Path $keyPath -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }
        
        foreach ($key in $keys) {
            if ($key.DisplayName -like "*Slack*" -and $key.PSChildName -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$') {
                Write-Host "Found MSI product code: $($key.PSChildName)"
                Write-Host "Attempting msiexec uninstall..."
                
                $msiArgs = @("/x", $key.PSChildName, "/qn", "/norestart", "REBOOT=ReallySuppress")
                $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -Wait -PassThru -NoNewWindow
                
                if ($process.ExitCode -eq 0) {
                    Write-Host "Successfully uninstalled via msiexec"
                    return $true
                } else {
                    Write-Host "msiexec returned exit code: $($process.ExitCode)"
                }
            }
        }
    }
    
    return $false
}

# Function to uninstall EXE-based installations
function Remove-SlackEXE {
    Write-Host "Checking for EXE-based Slack installations..."
    
    $uninstallKeys = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )
    
    $foundAny = $false
    
    foreach ($keyPath in $uninstallKeys) {
        $keys = Get-ChildItem -Path $keyPath -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }
        
        foreach ($key in $keys) {
            if ($key.DisplayName -like "*Slack*") {
                $foundAny = $true
                Write-Host "Found: $($key.DisplayName) at $($keyPath)"
                
                if ($key.UninstallString) {
                    # Extract the executable path and arguments
                    $uninstallString = $key.UninstallString
                    $exePath = ""
                    $arguments = ""
                    
                    if ($uninstallString -match '^"([^"]+)"(.*)') {
                        $exePath = $matches[1]
                        $arguments = $matches[2].Trim()
                    } elseif ($uninstallString -match '^([^\s]+)(.*)') {
                        $exePath = $matches[1]
                        $arguments = $matches[2].Trim()
                    }
                    
                    Write-Host "Uninstall executable: $exePath"
                    
                    # For Slack, common silent parameters
                    $silentParams = @(
                        "--uninstall --force-uninstall",
                        "--uninstall",
                        "/S",
                        "/SILENT",
                        "-s"
                    )
                    
                    # First try QuietUninstallString if available
                    if ($key.QuietUninstallString) {
                        Write-Host "Trying QuietUninstallString..."
                        $process = Start-Process -FilePath "cmd.exe" -ArgumentList "/c `"$($key.QuietUninstallString)`"" -Wait -PassThru -NoNewWindow
                        if ($process.ExitCode -eq 0) {
                            Write-Host "Successfully uninstalled using QuietUninstallString"
                            return $true
                        }
                    }
                    
                    # Try each silent parameter
                    foreach ($param in $silentParams) {
                        Write-Host "Trying with parameters: $param"
                        
                        try {
                            $fullArgs = if ($arguments) { "$arguments $param" } else { $param }
                            $process = Start-Process -FilePath $exePath -ArgumentList $fullArgs -Wait -PassThru -NoNewWindow -ErrorAction Stop
                            
                            if ($process.ExitCode -eq 0) {
                                Write-Host "Successfully uninstalled with parameters: $param"
                                return $true
                            } else {
                                Write-Host "Exit code: $($process.ExitCode)"
                            }
                        } catch {
                            Write-Host "Error: $_"
                        }
                    }
                }
            }
        }
    }
    
    if (-not $foundAny) {
        Write-Host "No EXE-based Slack installations found in registry"
    }
    
    return $false
}

# Function to kill Slack processes
function Stop-SlackProcesses {
    Write-Host "Checking for running Slack processes..."
    $processes = Get-Process -Name "Slack*" -ErrorAction SilentlyContinue
    
    if ($processes) {
        Write-Host "Found $($processes.Count) Slack process(es). Attempting to stop..."
        foreach ($proc in $processes) {
            try {
                $proc | Stop-Process -Force -ErrorAction Stop
                Write-Host "Stopped process: $($proc.ProcessName) (PID: $($proc.Id))"
            } catch {
                Write-Host "Failed to stop process: $($proc.ProcessName) - $_"
            }
        }
        Start-Sleep -Seconds 2
    }
}

# Function to clean up Slack folders
function Remove-SlackFolders {
    Write-Host "Cleaning up Slack folders..."
    
    $foldersToRemove = @(
        "$env:LOCALAPPDATA\Slack",
        "$env:APPDATA\Slack",
        "$env:ProgramFiles\Slack",
        "${env:ProgramFiles(x86)}\Slack"
    )
    
    foreach ($folder in $foldersToRemove) {
        if (Test-Path $folder) {
            Write-Host "Removing folder: $folder"
            try {
                Remove-Item -Path $folder -Recurse -Force -ErrorAction Stop
                Write-Host "Successfully removed: $folder"
            } catch {
                Write-Host "Failed to remove folder: $_"
            }
        }
    }
}

# Main uninstallation logic
try {
    # Stop Slack processes first
    Stop-SlackProcesses
    
    # Try MSI uninstallation first
    $msiResult = Remove-SlackMSI
    if ($msiResult) {
        $uninstalled = $true
        Write-Host "Slack uninstalled via MSI method"
    }
    
    # Try EXE uninstallation
    $exeResult = Remove-SlackEXE
    if ($exeResult) {
        $uninstalled = $true
        Write-Host "Slack uninstalled via EXE method"
    }
    
    # Clean up folders regardless of uninstall method success
    Remove-SlackFolders
    
    # Final verification
    Start-Sleep -Seconds 3
    $remainingInstalls = @(
        Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Slack*" }
        Get-ItemProperty "HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Slack*" }
        Get-ItemProperty "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Slack*" }
    )
    
    if ($remainingInstalls.Count -eq 0) {
        Write-Host "Verification: No Slack installations found in registry"
        $exitCode = 0
    } elseif ($uninstalled) {
        Write-Host "Warning: Some Slack registry entries remain, but uninstallation was attempted"
        $exitCode = 0
    } else {
        Write-Host "Error: Slack uninstallation failed"
        $exitCode = 1
    }
    
} catch {
    Write-Host "Critical error during uninstallation: $_"
    $exitCode = 1
}

Write-Host "Slack uninstallation script completed with exit code: $exitCode"
Exit $exitCode
