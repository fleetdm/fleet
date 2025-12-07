$softwareName = "MicrosoftCompanyPortal"
$zipPath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.zip"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"
$logFile = "$env:PUBLIC\install-log-$softwareName.txt"

$userScript = @"
`$ErrorActionPreference = "Stop"
`$zipPath = "$zipPath"
`$exitCodeFile = "$exitCodeFile"
`$logFile = "$logFile"
`$exitCode = 0
`$extractPath = "$env:TEMP\fleet-companyportal-extract"

# Redirect all output to log file and stdout
function Write-Log {
    param([string]`$Message)
    `$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    `$logMessage = "[`$timestamp] `$Message"
    Add-Content -Path `$logFile -Value `$logMessage -Force
    Write-Host `$Message
}

try {
    Write-Log "=== Company Portal Installation Start ==="
    Write-Log "Zip Path: `$zipPath"
    Write-Log "Extract Path: `$extractPath"

    # Verify zip file exists
    if (-not (Test-Path `$zipPath)) {
        throw "Zip file not found: `$zipPath"
    }
    Write-Log "Zip file exists"

    # Extract the zip file
    Write-Log "[1/4] Extracting zip file..."
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    New-Item -ItemType Directory -Path `$extractPath -Force | Out-Null
    
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::ExtractToDirectory(`$zipPath, `$extractPath)
    Write-Log "[1/4] Extraction complete"

    # Find the nested appx file dynamically
    Write-Log "Searching for appx/appxbundle files..."
    `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appxbundle" -ErrorAction SilentlyContinue
    if (-not `$appxFiles) {
        `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appx" -ErrorAction SilentlyContinue
    }
    
    if (-not `$appxFiles -or `$appxFiles.Count -eq 0) {
        throw "No appx or appxbundle file found in extracted zip"
    }
    
    `$appxPath = `$appxFiles[0].FullName
    Write-Log "Found appx file: `$appxPath"

    # Provision for all future users (this works in headless environments)
    Write-Log "[2/4] Provisioning for all future users..."
    try {
        `$provisionResult = Add-AppProvisionedPackage -Online -PackagePath `$appxPath -SkipLicense -ErrorAction Stop 2>&1
        Write-Log "Provisioning output: `$provisionResult"
        Write-Log "[2/4] Provisioning complete"
    } catch {
        Write-Log "ERROR: Provisioning failed: `$(`$_.Exception.Message)"
        throw
    }

    # Also install for current user so osquery can detect it immediately
    # Note: This may fail in headless/CI environments, but provisioning is sufficient
    Write-Log "[3/4] Installing for current user..."
    `$userInstallSuccess = `$false
    try {
        `$installResult = Add-AppxPackage -Path `$appxPath -ErrorAction Stop 2>&1
        Write-Log "Installation output: `$installResult"
        `$userInstallSuccess = `$true
        Write-Log "[3/4] Installation complete"
    } catch {
        Write-Log "WARNING: User installation failed (may be headless environment): `$(`$_.Exception.Message)"
        Write-Log "[3/4] Continuing anyway - package is provisioned for future users"
        # Don't fail the script if user install fails - provisioning is the important part
    }

    # Poll for package registration (up to 30 seconds)
    # In headless environments, the package may not show up in Get-AppxPackage
    # but it's still provisioned and will appear for future users
    Write-Log "[4/4] Polling for registration (max 30s)..."
    `$maxAttempts = 30
    `$attempt = 0
    `$installed = `$null

    while (`$attempt -lt `$maxAttempts) {
        `$installed = Get-AppxPackage -Name "Microsoft.CompanyPortal" -ErrorAction SilentlyContinue
        if (`$installed) {
            Write-Log "[4/4] Package registered after `$attempt seconds"
            break
        }
        Start-Sleep -Seconds 1
        `$attempt++
    }

    # Check if package is provisioned (works even if user install failed)
    `$provisioned = Get-AppxProvisionedPackage -Online | Where-Object { `$_.DisplayName -like "*Company Portal*" } | Select-Object -First 1
    if (`$provisioned) {
        Write-Log "Package is provisioned: `$(`$provisioned.DisplayName)"
    }

    if (-not `$installed -and -not `$provisioned) {
        Write-Log "[4/4] ERROR: Package not registered and not provisioned"
        `$exitCode = 1
    } elseif (-not `$installed) {
        # Package is provisioned but not installed for current user (likely headless environment)
        Write-Log "[4/4] Package is provisioned but not installed for current user (headless environment?)"
        Write-Log "=== Installation Successful (Provisioned) ==="
        Write-Log "Package will be available for future user logins"
        # Still consider this successful - provisioning is what matters
    } else {
        Write-Log "=== Installation Successful ==="
        Write-Log "Package: `$(`$installed.PackageFullName)"
        Write-Log "Version: `$(`$installed.Version)"
    }
    
    # Give osquery time to detect the app in the programs table
    Write-Log "Waiting for osquery to detect app (5 seconds)..."
    Start-Sleep -Seconds 5
} catch {
    Write-Log "=== Installation Failed ==="
    Write-Log "Error: `$(`$_.Exception.Message)"
    Write-Log "Stack: `$(`$_.ScriptStackTrace)"
    `$exitCode = 1
} finally {
    # Clean up extracted files
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Log "Exit Code: `$exitCode"
    Set-Content -Path `$exitCodeFile -Value `$exitCode -Force
}

Exit `$exitCode
"@

$exitCode = 0

try {
    Write-Host "=== Company Portal Install Wrapper Start ==="
    Write-Host "Zip Path: $zipPath"
    Write-Host "Script Path: $scriptPath"
    Write-Host "Exit Code File: $exitCodeFile"

    # Wait for an interactive user to be logged on
    Write-Host "Waiting for interactive user..."
    $maxWaitAttempts = 24  # 2 minutes max
    $waitAttempt = 0
    $userName = $null
    
    while ($waitAttempt -lt $maxWaitAttempts) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            Write-Host "Found interactive user: $userName"
            break
        } else {
            Start-Sleep -Seconds 5
            $waitAttempt++
        }
    }
    
    if (-not $userName -or -not ($userName -like "*\*")) {
        throw "Timeout waiting for interactive user after $($maxWaitAttempts * 5) seconds"
    }

    # Write the install script to disk
    Write-Host "Writing install script to disk..."
    Set-Content -Path $scriptPath -Value $userScript -Force
    Write-Host "Script written successfully"

    # Build task action: run script (output is captured via Write-Log function)
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""

    $trigger = New-ScheduledTaskTrigger -AtLogOn

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

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
    Write-Host "Task started successfully"

    # Wait for it to complete
    Write-Host "Waiting for task to complete..."
    while ($state -eq "Running") {
        Start-Sleep -Seconds 5
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 300) { throw "Timeout waiting for task to finish." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }
    Write-Host "Task completed with state: $state"

    if (Test-Path $exitCodeFile) {
        $exitCode = [int](Get-Content $exitCodeFile)
        Write-Host "Exit code from task: $exitCode"
    } else {
        Write-Host "WARNING: Exit code file not found, assuming failure"
        $exitCode = 1
    }
    
    # Read and display the log file (last 100 lines to avoid truncation)
    if (Test-Path $logFile) {
        Write-Host "=== Task Output Log (last 100 lines) ==="
        $logLines = Get-Content $logFile
        $lineCount = $logLines.Count
        if ($lineCount -gt 100) {
            Write-Host "... (showing last 100 of $lineCount lines) ..."
            $logLines[-100..-1] | ForEach-Object { Write-Host $_ }
        } else {
            $logLines | ForEach-Object { Write-Host $_ }
        }
        Write-Host "=== End Task Output Log ==="
    } else {
        Write-Host "WARNING: Log file not found at $logFile"
    }

} catch {
    Write-Host "=== Wrapper Error ==="
    Write-Host "Error: $_"
    Write-Host "Stack: $($_.ScriptStackTrace)"
    $exitCode = 1
} finally {
    # Clean up
    Write-Host "Cleaning up scheduled task and files..."
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
    # Keep log file for debugging, but could remove it: Remove-Item -Path $logFile -Force -ErrorAction SilentlyContinue
    Write-Host "=== Company Portal Install Wrapper End ==="
}

Exit $exitCode

