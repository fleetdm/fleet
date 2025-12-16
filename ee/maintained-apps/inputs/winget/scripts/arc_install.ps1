$softwareName = "Arc"
$msixPath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.msix"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

$userScript = @"
`$msixPath = "$msixPath"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0

try {
    Write-Host "=== Arc Installation Start ==="
    Write-Host "MSIX Path: `$msixPath"

    # Provision for all future users
    Write-Host "[1/3] Provisioning for all future users..."
    Add-AppProvisionedPackage -Online -PackagePath `$msixPath -SkipLicense -ErrorAction Stop
    Write-Host "[1/3] Provisioning complete"

    # Also install for current user so osquery can detect it immediately
    Write-Host "[2/3] Installing for current user..."
    try {
        Add-AppxPackage -Path `$msixPath -ErrorAction Stop | Out-Null
        Write-Host "[2/3] Installation complete"
    } catch {
        Write-Host "[2/3] User installation failed (may be headless environment), continuing..."
        # Don't fail the script if user install fails - provisioning is the important part
    }

    # Poll for package registration (up to 30 seconds)
    Write-Host "[3/3] Polling for registration (max 30s)..."
    `$maxAttempts = 30
    `$attempt = 0
    `$installed = `$null

    while (`$attempt -lt `$maxAttempts) {
        `$installed = Get-AppxPackage -Name "Arc" -ErrorAction SilentlyContinue
        if (`$installed) {
            Write-Host "[3/3] Package registered after `$attempt seconds"
            break
        }
        Start-Sleep -Seconds 1
        `$attempt++
    }

    # Check if package is provisioned (works even if user install failed)
    `$provisioned = Get-AppxProvisionedPackage -Online | Where-Object { `$_.DisplayName -eq "Arc" } | Select-Object -First 1
    if (-not `$installed -and -not `$provisioned) {
        Write-Host "[3/3] ERROR: Package not registered and not provisioned"
        `$exitCode = 1
    } elseif (-not `$installed) {
        # Package is provisioned but not installed for current user (likely headless environment)
        Write-Host "[3/3] Package is provisioned but not installed for current user"
        Write-Host "=== Installation Successful (Provisioned) ==="
        Write-Host "Package: `$(`$provisioned.DisplayName)"
        Write-Host "Version: `$(`$provisioned.Version)"
    } else {
        Write-Host "=== Installation Successful ==="
        Write-Host "Package: `$(`$installed.PackageFullName)"
        Write-Host "Version: `$(`$installed.Version)"
    }
    
    # Give osquery time to detect the app in the programs table
    Start-Sleep -Seconds 5
} catch {
    Write-Host "=== Installation Failed ==="
    Write-Host "Error: `$(`$_.Exception.Message)"
    `$exitCode = 1
} finally {
    Write-Host "Exit Code: `$exitCode"
    Set-Content -Path `$exitCodeFile -Value `$exitCode
}

Exit `$exitCode
"@

$exitCode = 0

try {
    # Wait for an interactive user to be logged on (with timeout for headless environments)
    $maxWaitTime = 60  # Maximum wait time in seconds
    $startTime = Get-Date
    $userName = $null
    
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            break
        }
        
        $elapsed = (New-Timespan -Start $startTime).TotalSeconds
        if ($elapsed -gt $maxWaitTime) {
            # Timeout reached - likely headless environment
            # Run installation directly without scheduled task
            Write-Host "No interactive user detected after $maxWaitTime seconds. Running installation directly (headless mode)..."
            
            # Write the script to disk and execute it directly
            Set-Content -Path $scriptPath -Value $userScript -Force
            & powershell.exe -WindowStyle Hidden -ExecutionPolicy Bypass -File $scriptPath
            
            if (Test-Path $exitCodeFile) {
                $exitCode = Get-Content $exitCodeFile
            } else {
                $exitCode = 1
            }
            
            # Clean up
            Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
            Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
            
            Exit $exitCode
        }
        
        Start-Sleep -Seconds 5
    }

    # Write the install script to disk
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Build task action: run script (output goes to stdout for Fleet)
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""

    $trigger = New-ScheduledTaskTrigger -AtLogOn

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null

    # Start the task
    Start-ScheduledTask -TaskName $taskName

    # Wait for it to start
    $startDate = Get-Date
    $state = (Get-ScheduledTask -TaskName $taskName).State
    while ($state -ne "Running") {
        Start-Sleep -Seconds 1
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 120) { throw "Timeout waiting for task to start." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }

    # Wait for it to complete
    while ($state -eq "Running") {
        Start-Sleep -Seconds 5
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 120) { throw "Timeout waiting for task to finish." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }

    if (Test-Path $exitCodeFile) {
        $exitCode = Get-Content $exitCodeFile
    } else {
        $exitCode = 1
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Clean up
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
}

Exit $exitCode

