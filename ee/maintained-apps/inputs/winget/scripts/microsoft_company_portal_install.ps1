$softwareName = "MicrosoftCompanyPortal"
$zipPath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.zip"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

$userScript = @"
`$zipPath = "$zipPath"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0
`$extractPath = "$env:TEMP\fleet-companyportal-extract"

try {
    Write-Host "=== Company Portal Installation Start ==="
    Write-Host "Zip Path: `$zipPath"

    # Extract the zip file
    Write-Host "[1/4] Extracting zip file..."
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    New-Item -ItemType Directory -Path `$extractPath -Force | Out-Null
    
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::ExtractToDirectory(`$zipPath, `$extractPath)
    Write-Host "[1/4] Extraction complete"

    # Find the nested appx file
    `$appxPath = Join-Path `$extractPath "c797dbb4414543f59d35e59e5225824e.appxbundle"
    if (-not (Test-Path `$appxPath)) {
        throw "Nested appx file not found: `$appxPath"
    }
    Write-Host "Found nested appx file: `$appxPath"

    # Provision for all future users
    Write-Host "[2/4] Provisioning for all future users..."
    try {
        $provisionResult = Add-AppProvisionedPackage -Online -PackagePath `$appxPath -SkipLicense -ErrorAction Stop 2>&1
        Write-Host "[2/4] Provisioning complete"
    } catch {
        Write-Host "[2/4] Provisioning error: `$(`$_.Exception.Message)"
        throw
    }

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
    `$exitCode = 1
} finally {
    # Clean up extracted files
    if (Test-Path `$extractPath) {
        Remove-Item -Path `$extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Host "Exit Code: `$exitCode"
    Set-Content -Path `$exitCodeFile -Value `$exitCode
}

Exit `$exitCode
"@

$exitCode = 0

try {
    # Wait for an interactive user to be logged on
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            break
        } else {
            Start-Sleep -Seconds 5
        }
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

