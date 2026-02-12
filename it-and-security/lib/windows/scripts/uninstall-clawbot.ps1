# Clawbot/moltbot Uninstall Script for Windows
# Removes Clawbot (also known as moltbot) and cleans up related files

$softwareName = "Clawbot/moltbot"
$exitCode = 0
$uninstalled = $false

Write-Host "Starting $softwareName uninstallation process..."

# Function to stop Clawbot/moltbot processes
function Stop-ClawbotProcesses {
    Write-Host "Checking for running Clawbot/moltbot processes..."
    $processNames = @("clawbot", "moltbot", "Clawbot", "Moltbot")
    foreach ($procName in $processNames) {
        $processes = Get-Process -Name $procName -ErrorAction SilentlyContinue
        if ($processes) {
            Write-Host "Found $($processes.Count) $procName process(es). Stopping..."
            foreach ($proc in $processes) {
                try {
                    $proc | Stop-Process -Force -ErrorAction Stop
                    Write-Host "Stopped process: $($proc.ProcessName) (PID: $($proc.Id))"
                } catch {
                    Write-Host "Failed to stop process: $($proc.ProcessName) - $_"
                }
            }
        }
    }
    Start-Sleep -Seconds 2
}

# Function to stop and remove services
function Remove-ClawbotServices {
    Write-Host "Checking for Clawbot/moltbot services..."
    $serviceNames = @("clawbot", "moltbot", "Clawbot", "Moltbot")
    foreach ($svcName in $serviceNames) {
        $service = Get-Service -Name $svcName -ErrorAction SilentlyContinue
        if ($service) {
            Write-Host "Found service: $svcName. Stopping and removing..."
            try {
                Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
                sc.exe delete $svcName | Out-Null
                Write-Host "Removed service: $svcName"
            } catch {
                Write-Host "Error removing service $svcName : $_"
            }
        }
    }
}

# Function to uninstall via registry uninstall strings
function Remove-ClawbotFromRegistry {
    Write-Host "Checking for Clawbot/moltbot in registry..."

    $uninstallKeys = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )

    foreach ($keyPath in $uninstallKeys) {
        $keys = Get-ChildItem -Path $keyPath -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

        foreach ($key in $keys) {
            if ($key.DisplayName -like "*Clawbot*" -or $key.DisplayName -like "*moltbot*") {
                Write-Host "Found: $($key.DisplayName)"

                # Try QuietUninstallString first
                if ($key.QuietUninstallString) {
                    Write-Host "Running QuietUninstallString..."
                    $process = Start-Process -FilePath "cmd.exe" -ArgumentList "/c `"$($key.QuietUninstallString)`"" -Wait -PassThru -NoNewWindow
                    if ($process.ExitCode -eq 0) {
                        Write-Host "Successfully uninstalled via QuietUninstallString"
                        return $true
                    }
                }

                # Try UninstallString with silent flags
                if ($key.UninstallString) {
                    $uninstallString = $key.UninstallString
                    $exePath = ""
                    if ($uninstallString -match '^"([^"]+)"(.*)') {
                        $exePath = $matches[1]
                    } elseif ($uninstallString -match '^([^\s]+)(.*)') {
                        $exePath = $matches[1]
                    }

                    if ($exePath -and (Test-Path $exePath)) {
                        foreach ($param in @("/S", "/SILENT", "--uninstall", "/qn")) {
                            Write-Host "Trying uninstall with: $param"
                            try {
                                $process = Start-Process -FilePath $exePath -ArgumentList $param -Wait -PassThru -NoNewWindow -ErrorAction Stop
                                if ($process.ExitCode -eq 0) {
                                    Write-Host "Successfully uninstalled with: $param"
                                    return $true
                                }
                            } catch {
                                Write-Host "Error: $_"
                            }
                        }
                    }
                }

                # Try MSI uninstall if product code is a GUID
                if ($key.PSChildName -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$') {
                    Write-Host "Attempting MSI uninstall for product code: $($key.PSChildName)"
                    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList @("/x", $key.PSChildName, "/qn", "/norestart") -Wait -PassThru -NoNewWindow
                    if ($process.ExitCode -eq 0) {
                        Write-Host "Successfully uninstalled via MSI"
                        return $true
                    }
                }
            }
        }
    }

    return $false
}

# Function to clean up Clawbot/moltbot folders
function Remove-ClawbotFolders {
    Write-Host "Cleaning up Clawbot/moltbot folders..."

    $foldersToRemove = @(
        "$env:ProgramFiles\Clawbot",
        "$env:ProgramFiles\Moltbot",
        "${env:ProgramFiles(x86)}\Clawbot",
        "${env:ProgramFiles(x86)}\Moltbot",
        "$env:LOCALAPPDATA\Clawbot",
        "$env:LOCALAPPDATA\Moltbot",
        "$env:APPDATA\Clawbot",
        "$env:APPDATA\Moltbot",
        "$env:ProgramData\Clawbot",
        "$env:ProgramData\Moltbot"
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
    Stop-ClawbotProcesses
    Remove-ClawbotServices

    $registryResult = Remove-ClawbotFromRegistry
    if ($registryResult) {
        $uninstalled = $true
        Write-Host "Clawbot/moltbot uninstalled via registry method"
    }

    Remove-ClawbotFolders

    # Final verification
    Start-Sleep -Seconds 3
    $remainingInstalls = @(
        Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Clawbot*" -or $_.DisplayName -like "*moltbot*" }
        Get-ItemProperty "HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Clawbot*" -or $_.DisplayName -like "*moltbot*" }
        Get-ItemProperty "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like "*Clawbot*" -or $_.DisplayName -like "*moltbot*" }
    )

    if ($remainingInstalls.Count -eq 0) {
        Write-Host "Verification: No Clawbot/moltbot installations found in registry"
        $exitCode = 0
    } elseif ($uninstalled) {
        Write-Host "Warning: Some registry entries remain, but uninstallation was attempted"
        $exitCode = 0
    } else {
        Write-Host "Error: Clawbot/moltbot uninstallation failed"
        $exitCode = 1
    }

} catch {
    Write-Host "Critical error during uninstallation: $_"
    $exitCode = 1
}

Write-Host "Clawbot/moltbot uninstallation script completed with exit code: $exitCode"
Exit $exitCode
