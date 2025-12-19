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

    # Verify zip file exists
    if (-not (Test-Path `$zipPath)) {
        throw "Zip file not found: `$zipPath"
    }

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
    `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appxbundle" -ErrorAction SilentlyContinue
    if (-not `$appxFiles) {
        `$appxFiles = Get-ChildItem -Path `$extractPath -Recurse -Filter "*.appx" -ErrorAction SilentlyContinue
    }
    
    if (-not `$appxFiles -or `$appxFiles.Count -eq 0) {
        throw "No appx or appxbundle file found in extracted zip"
    }
    
    `$appxPath = `$appxFiles[0].FullName
    Write-Host "Found appx file: `$appxPath"

    # Provision for all future users (this works in headless environments)
    Write-Host "[2/4] Provisioning for all future users..."
    Add-AppProvisionedPackage -Online -PackagePath `$appxPath -SkipLicense -ErrorAction Stop
    Write-Host "[2/4] Provisioning complete"

    # Also install for current user so osquery can detect it immediately
    # Note: This may fail in headless/CI environments, but provisioning is sufficient
    Write-Host "[3/4] Installing for current user..."
    try {
        Add-AppxPackage -Path `$appxPath -ErrorAction Stop | Out-Null
        Write-Host "[3/4] Installation complete"
    } catch {
        Write-Host "[3/4] User installation failed (may be headless environment), continuing..."
        # Don't fail the script if user install fails - provisioning is the important part
    }

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

    # Check if package is provisioned (works even if user install failed)
    `$provisioned = Get-AppxProvisionedPackage -Online | Where-Object { `$_.DisplayName -eq "Microsoft.CompanyPortal" } | Select-Object -First 1
    if (-not `$installed -and -not `$provisioned) {
        Write-Host "[4/4] ERROR: Package not registered and not provisioned"
        `$exitCode = 1
    } elseif (-not `$installed) {
        # Package is provisioned but not installed for current user (likely headless environment)
        Write-Host "[4/4] Package is provisioned but not installed for current user"
        Write-Host "=== Installation Successful (Provisioned) ==="
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

    # Build task action: run script (output is captured via Write-Log function)
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
        if ($elapsed -gt 300) { throw "Timeout waiting for task to finish." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }

    if (Test-Path $exitCodeFile) {
        $exitCode = [int](Get-Content $exitCodeFile)
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

