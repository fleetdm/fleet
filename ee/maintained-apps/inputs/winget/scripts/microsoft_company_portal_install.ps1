$softwareName = "MicrosoftCompanyPortal"
$zipPath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.zip"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

$userScript = @"
`$ErrorActionPreference = "Stop"
`$zipPath = "$zipPath"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0
`$extractPath = "$env:TEMP\fleet-companyportal-extract"

try {
    Write-Host "=== Company Portal Installation Start ==="
    Write-Host "Zip Path: `$zipPath"
    Write-Host "Extract Path: `$extractPath"

    # Verify zip file exists
    if (-not (Test-Path `$zipPath)) {
        throw "Zip file not found: `$zipPath"
    }
    Write-Host "Zip file exists"

    # Extract the zip file
    Write-Host "[1/4] Extracting zip file..."
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    New-Item -ItemType Directory -Path `$extractPath -Force | Out-Null
    
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::ExtractToDirectory(`$zipPath, `$extractPath)
    Write-Host "[1/4] Extraction complete"

    # Find the nested appx file dynamically
    Write-Host "Searching for appx/appxbundle files..."
    `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appxbundle" -ErrorAction SilentlyContinue
    if (-not `$appxFiles) {
        `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appx" -ErrorAction SilentlyContinue
    }
    
    if (-not `$appxFiles -or `$appxFiles.Count -eq 0) {
        throw "No appx or appxbundle file found in extracted zip"
    }
    
    `$appxPath = `$appxFiles[0].FullName
    Write-Host "Found appx file: `$appxPath"

    # Provision for all future users
    Write-Host "[2/4] Provisioning for all future users..."
    Add-AppProvisionedPackage -Online -PackagePath `$appxPath -SkipLicense -ErrorAction Stop
    Write-Host "[2/4] Provisioning complete"

    # Also install for current user so osquery can detect it immediately
    Write-Host "[3/4] Installing for current user..."
    Add-AppxPackage -Path `$appxPath -ErrorAction Stop
    Write-Host "[3/4] Installation complete"

    # Poll for package registration (up to 30 seconds)
    Write-Host "[4/4] Polling for registration (max 30s)..."
    `$maxAttempts = 30
    `$attempt = 0
    `$installed = `$null

    while (`$attempt -lt `$maxAttempts) {
        `$installed = Get-AppxPackage -Name "Microsoft.CompanyPortal" -ErrorAction SilentlyContinue
        if (`$installed) {
            Write-Host "[4/4] Package registered after `$attempt seconds"
            break
        }
        Start-Sleep -Seconds 1
        `$attempt++
    }

    if (-not `$installed) {
        Write-Host "[4/4] ERROR: Package not registered after `$attempt seconds"
        `$exitCode = 1
    } else {
        Write-Host "=== Installation Successful ==="
        Write-Host "Package: `$(`$installed.PackageFullName)"
        Write-Host "Version: `$(`$installed.Version)"
        
        # Give osquery time to detect the app in the programs table
        Write-Host "Waiting for osquery to detect app (5 seconds)..."
        Start-Sleep -Seconds 5
    }
} catch {
    Write-Host "=== Installation Failed ==="
    Write-Host "Error: `$(`$_.Exception.Message)"
    Write-Host "Stack: `$(`$_.ScriptStackTrace)"
    `$exitCode = 1
} finally {
    # Clean up extracted files
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Host "Exit Code: `$exitCode"
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

    # Build task action: run script (output goes to stdout for Fleet)
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
    Write-Host "=== Company Portal Install Wrapper End ==="
}

Exit $exitCode

