$softwareName = "Postman"
$productKey = "Postman"
$taskName = "fleet-uninstall-$softwareName"
$scriptPath = "$env:PUBLIC\uninstall-$softwareName.ps1"
$logFile = "$env:PUBLIC\uninstall-output-$softwareName.txt"
$exitCodeFile = "$env:PUBLIC\uninstall-exitcode-$softwareName.txt"

# Embedded uninstall script
$userScript = @"
`$userKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\$productKey"
`$exitCode = 0
`$logFile = "$logFile"
`$exitCodeFile = "$exitCodeFile"

Start-Transcript -Path "$logFile" -Append

try {
    `$key = Get-ItemProperty -Path `$userKey -ErrorAction Stop

    `$uninstallCommand = if (`$key.QuietUninstallString) {
        `$key.QuietUninstallString
    } else {
        `$key.UninstallString
    }

    `$splitArgs = `$uninstallCommand.Split('"')
    if (`$splitArgs.Length -gt 1) {
        if (`$splitArgs.Length -eq 3) {
            `$uninstallArgs = `$splitArgs[2].Trim()
        } elseif (`$splitArgs.Length -gt 3) {
            Throw "Uninstall command contains multiple quoted strings. Please update the uninstall script.`nUninstall command: `$uninstallCommand"
        }
        `$uninstallCommand = `$splitArgs[1]
    }

    Write-Host "Uninstall command: `$uninstallCommand"
    Write-Host "Uninstall args: `$uninstallArgs"

    `$processOptions = @{
        FilePath = `$uninstallCommand
        PassThru = `$true
        Wait     = `$true
    }

    if (`$uninstallArgs -ne '') {
        `$processOptions.ArgumentList = "`$uninstallArgs"
    }

    `$process = Start-Process @processOptions
    `$exitCode = `$process.ExitCode
    Write-Host "Uninstall exit code: `$exitCode"
}
catch {
    Write-Host "Error: `$_.Exception.Message"
    `$exitCode = 1
}
finally {
    Set-Content -Path `$exitCodeFile -Value `$exitCode
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
