$softwareName = "MSTeams-x64"
$msixPath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.msix"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$logFile = "$env:PUBLIC\install-output-$softwareName.txt"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

$userScript = @"
`$msixPath = "$msixPath"
`$logFile = "$logFile"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0

Start-Transcript -Path `$logFile -Append

Write-Host "=== Teams MSIX Installation Script ==="
Write-Host "MSIX Path: `$msixPath"
Write-Host "Current User: `$env:USERNAME"
Write-Host "Start Time: `$(Get-Date)"
Write-Host ""

try {
    # Provision for all future users
    Write-Host "[Step 1/3] Provisioning package for all future users..."
    Add-AppProvisionedPackage -Online -PackagePath `$msixPath -SkipLicense -Verbose
    Write-Host "[Step 1/3] SUCCESS: Package provisioned"
    Write-Host ""

    # Also install for current user so osquery can detect it immediately
    Write-Host "[Step 2/3] Installing package for current user..."
    Add-AppxPackage -Path `$msixPath -Verbose
    Write-Host "[Step 2/3] SUCCESS: Package installed for current user"
    Write-Host ""

    # Verify installation
    Write-Host "[Step 3/3] Verifying installation..."
    `$installed = Get-AppxPackage -Name "MSTeams"
    if (`$installed) {
        Write-Host "[Step 3/3] SUCCESS: Teams package found after installation"
        Write-Host "  Name: `$(`$installed.Name)"
        Write-Host "  Version: `$(`$installed.Version)"
        Write-Host "  PackageFullName: `$(`$installed.PackageFullName)"
        Write-Host "  PackageFamilyName: `$(`$installed.PackageFamilyName)"
        Write-Host "  InstallLocation: `$(`$installed.InstallLocation)"
    } else {
        Write-Host "[Step 3/3] ERROR: Teams package NOT found after installation!"
        `$exitCode = 1
    }
} catch {
    Write-Host "ERROR OCCURRED:"
    Write-Host "  Message: `$(`$_.Exception.Message)"
    if (`$_.Exception.InnerException) {
        Write-Host "  Inner Exception: `$(`$_.Exception.InnerException.Message)"
    }
    Write-Host "  Stack Trace: `$(`$_.ScriptStackTrace)"
    `$exitCode = 1
} finally {
    Write-Host ""
    Write-Host "End Time: `$(Get-Date)"
    Write-Host "Exit Code: `$exitCode"
    Set-Content -Path `$exitCodeFile -Value `$exitCode
}

Stop-Transcript

Exit `$exitCode
"@

$exitCode = 0

try {
    Write-Host "=== Teams Installation Orchestrator ==="
    Write-Host "Installer Path: $msixPath"
    Write-Host ""

    # Wait for an interactive user to be logged on
    Write-Host "Waiting for interactive user..."
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            Write-Host "Interactive user detected: $userName"
            break
        } else {
            Write-Host "No interactive user yet, waiting..."
            Start-Sleep -Seconds 5
        }
    }

    # Write the install script to disk
    Write-Host "Creating installation script at: $scriptPath"
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Build task action: run script, redirect stdout/stderr to log file
    Write-Host "Configuring scheduled task..."
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`" *> `"$logFile`" 2>&1"

    $trigger = New-ScheduledTaskTrigger -AtLogOn

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

    Write-Host "Registering scheduled task: $taskName"
    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null

    # Start the task
    Write-Host "Starting scheduled task..."
    Start-ScheduledTask -TaskName $taskName

    # Wait for it to start
    Write-Host "Waiting for task to start..."
    $startDate = Get-Date
    $state = (Get-ScheduledTask -TaskName $taskName).State
    while ($state -ne "Running") {
        Start-Sleep -Seconds 1
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 120) { throw "Timeout waiting for task to start." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }
    Write-Host "Task is running..."

    # Wait for it to complete
    while ($state -eq "Running") {
        Start-Sleep -Seconds 5
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 120) { throw "Timeout waiting for task to finish." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }
    Write-Host "Task completed with state: $state"

    # Show task output
    Write-Host "`n=== Scheduled Task Output ==="
    if (Test-Path $logFile) {
        Get-Content $logFile | Write-Host
    } else {
        Write-Host "WARNING: Log file not found at: $logFile"
    }
    Write-Host "=== End Scheduled Task Output ===`n"

    if (Test-Path $exitCodeFile) {
        $exitCode = Get-Content $exitCodeFile
        Write-Host "Scheduled task exit code: $exitCode"
    } else {
        Write-Host "WARNING: Exit code file not found at: $exitCodeFile"
        $exitCode = 1
    }

    # Final verification
    Write-Host "`nFinal verification: Checking if Teams is installed..."
    $finalCheck = Get-AppxPackage -Name "MSTeams" -ErrorAction SilentlyContinue
    if ($finalCheck) {
        Write-Host "SUCCESS: Teams package found"
        Write-Host "  Version: $($finalCheck.Version)"
        Write-Host "  PackageFamilyName: $($finalCheck.PackageFamilyName)"
    } else {
        Write-Host "ERROR: Teams package NOT found in Get-AppxPackage"
        $exitCode = 1
    }

} catch {
    Write-Host "`nERROR in outer script: $_"
    Write-Host "Stack Trace: $($_.ScriptStackTrace)"
    $exitCode = 1
} finally {
    # Clean up
    Write-Host "`nCleaning up temporary files..."
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
    # TEMPORARY: Don't delete log files so they can be uploaded as GitHub Actions artifacts for debugging
    # Remove-Item -Path $logFile -Force -ErrorAction SilentlyContinue
    # Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
    Write-Host "Cleanup complete (logs preserved at $logFile for debugging)"
}

Exit $exitCode