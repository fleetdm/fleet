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

try {
    Add-AppProvisionedPackage -Online -PackagePath `$msixPath -SkipLicense
} catch {
    Write-Host "Error: `$_.Exception.Message"
    `$exitCode = 1
} finally {
    Set-Content -Path `$exitCodeFile -Value `$
}

Stop-Transcript

Exit `$exitCode
"@

$exitCode = 0

try {
    # Wait for an interactive user to be logged on
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            Write-Output "Interactive user detected: $userName"
            break
        } else {
            Start-Sleep -Seconds 5
        }
    }

    # Write the uninstall script to disk
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Build task action: run script, redirect stdout/stderr to log file
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`" *> `"$logFile`" 2>&1"

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

    # Show task output
    if (Test-Path $logFile) {
        Write-Host "`n--- Scheduled Task Output ---"
        Get-Content $logFile | Write-Host
    }

    if (Test-Path $exitCodeFile) {
        $exitCode = Get-Content $exitCodeFile
        Write-Host "`nScheduled task exit code: $exitCode"
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Clean up
    Write-Host "Cleaning up..."
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $logFile -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
}

Exit $exitCode