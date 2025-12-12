$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {

# Copy the installer to a public folder so that all can access it
# users
$exeFilename = Split-Path $exeFilePath -leaf
Copy-Item -Path $exeFilePath -Destination "${env:PUBLIC}" -Force
$exeFilePath = "${env:PUBLIC}\$exeFilename"

# Task properties. The task will be started by the logged in user
$action = New-ScheduledTaskAction -Execute "$exeFilePath"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Invoke-CimMethod -MethodName getowner).User
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

# Create a task object with the properties defined above
$task = New-ScheduledTask -Action $action -Trigger $trigger `
    -Settings $settings

# Register the task
$taskName = "fleet-install-$exeFilename"
Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

# keep track of the start time to cancel if taking too long to start
$startDate = Get-Date

# Start the task now that it is ready
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

# Wait for the task to be running
$state = (Get-ScheduledTask -TaskName "$taskName").State
Write-Host "ScheduledTask is '$state'"

while ($state  -ne "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting to run .exe..."

    $endDate = Get-Date
    $elapsedTime = New-Timespan -Start $startDate -End $endDate
    if ($elapsedTime.TotalSeconds -gt 120) {
        Throw "Timed-out waiting for scheduled task state."
    }

    Start-Sleep -Seconds 1
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

# Wait for the task to be done
$state = (Get-ScheduledTask -TaskName "$taskName").State
while ($state  -eq "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting for .exe to complete..."

    $endDate = Get-Date
    $elapsedTime = New-Timespan -Start $startDate -End $endDate
    if ($elapsedTime.TotalSeconds -gt 120) {
        Throw "Timed-out waiting for scheduled task state."
    }

    Start-Sleep -Seconds 10
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

# Remove task
Write-Host "Removing ScheduledTask: $taskName."
Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Remove installer
    Remove-Item -Path $exeFilePath -Force
}

Exit $exitCode