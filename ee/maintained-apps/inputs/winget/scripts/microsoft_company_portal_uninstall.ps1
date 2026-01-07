$softwareName = "MicrosoftCompanyPortal"
$packageName = "Microsoft.CompanyPortal"
$taskName = "fleet-uninstall-$softwareName.zip"
$scriptPath = "$env:PUBLIC\uninstall-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\uninstall-exitcode-$softwareName.txt"

$userScript = @"
`$packageName = "$packageName"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0

try {
    Write-Host "=== Company Portal Uninstallation Start ==="

    # Remove for current user
    Write-Host "Removing package for current user..."
    `$package = Get-AppxPackage -Name `$packageName -ErrorAction SilentlyContinue
    if (`$package) {
        Remove-AppxPackage -Package `$package.PackageFullName -ErrorAction Stop
        Write-Host "Removed for current user"
    } else {
        Write-Host "Package not found for current user"
    }

    # Also remove provisioned package for all future users
    Write-Host "Removing provisioned package for all future users..."
    `$provisioned = Get-AppxProvisionedPackage -Online | Where-Object { `$_.DisplayName -eq `$packageName } | Select-Object -First 1
    if (`$provisioned) {
        Remove-AppxProvisionedPackage -Online -PackageName `$provisioned.PackageName -ErrorAction Stop
        Write-Host "Removed provisioned package"
    } else {
        Write-Host "Provisioned package not found"
    }

    Write-Host "=== Uninstallation Successful ==="
} catch {
    Write-Host "=== Uninstallation Failed ==="
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
    # Wait for an interactive user to be logged on
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            break
        } else {
            Start-Sleep -Seconds 5
        }
    }

    # Write the uninstall script to disk
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Build task action: run script (output goes to stdout for Fleet)
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""

    $trigger = New-ScheduledTaskTrigger -AtLogOn

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force

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

