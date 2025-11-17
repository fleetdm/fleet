$softwareName = "figma"
$arguments = "-s"
$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-$softwareName.msix"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$logFile = "$env:PUBLIC\install-output-$softwareName.txt"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

$userScript = @"
`$exeFilePath = "$exeFilePath"
`$arguments = "$arguments"
`$logFile = "$logFile"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0
Start-Transcript -Path `$logFile -Append
try {
    `$exeFilename = Split-Path `$exeFilePath -leaf
    `$exePath = "`${env:PUBLIC}\`$exeFilename"
    & `$exePath `$arguments
    `$exitCode = `$LASTEXITCODE
    if (`$exitCode -eq 0 -or `$exitCode -eq $null) {
        `$exitCode = 0
    }
} catch {
    Write-Host "Error: `$_.Exception.Message"
    `$exitCode = 1
} finally {
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

    # Write the install script to disk
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Copy the installer to a public folder so that all users can access it
    $exeFilename = Split-Path $exeFilePath -leaf
    Copy-Item -Path $exeFilePath -Destination "${env:PUBLIC}" -Force
    $exeFilePath = "${env:PUBLIC}\$exeFilename"

    # Build task action: run script, redirect stdout/stderr to log file
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`" *> `"$logFile`" 2>&1"

    # Create a trigger that runs immediately (one-time trigger)
    $trigger = New-ScheduledTaskTrigger -Once -At (Get-Date)

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force

    # Start the task immediately
    Start-ScheduledTask -TaskName $taskName -TaskPath "\"

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
        $taskExitCode = Get-Content $exitCodeFile
        Write-Host "`nScheduled task exit code: $taskExitCode"
        if ($taskExitCode -ne "0" -and $taskExitCode -ne "") {
            $exitCode = [int]$taskExitCode
            throw "Scheduled task failed with exit code: $taskExitCode"
        }
        # Wait a moment for registry to update after installation
        Start-Sleep -Seconds 2
    } else {
        Write-Host "`nWarning: Exit code file not found. Installation may have failed."
        $exitCode = 1
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

