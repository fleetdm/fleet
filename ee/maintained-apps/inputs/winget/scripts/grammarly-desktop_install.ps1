# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Grammarly installs per user into %LOCALAPPDATA%, so it has to run as the
# logged-in user rather than as SYSTEM; this runs the installer through a
# scheduled task in that user's session. Nobody logged in means no install.

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {

$userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Invoke-CimMethod -MethodName getowner).User
if (-not $userName) {
    Write-Host "No logged-in user found. Grammarly installs per user and cannot be installed now."
    Exit 1
}

# Copy the installer to a public folder so that the logged-in user can read it
$exeFilename = Split-Path $exeFilePath -leaf
Copy-Item -Path $exeFilePath -Destination "${env:PUBLIC}" -Force
$exeFilePath = "${env:PUBLIC}\$exeFilename"

# /S is the NSIS silent switch
$action = New-ScheduledTaskAction -Execute "$exeFilePath" -Argument "/S"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
$task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings

$taskName = "fleet-install-$exeFilename"
Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

$startDate = Get-Date
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

$state = (Get-ScheduledTask -TaskName "$taskName").State
while ($state -ne "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting to run .exe..."

    if ((New-Timespan -Start $startDate -End (Get-Date)).TotalSeconds -gt 120) {
        Throw "Timed-out waiting for scheduled task state."
    }

    Start-Sleep -Seconds 1
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

$state = (Get-ScheduledTask -TaskName "$taskName").State
while ($state -eq "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting for .exe to complete..."

    if ((New-Timespan -Start $startDate -End (Get-Date)).TotalSeconds -gt 600) {
        Throw "Timed-out waiting for scheduled task to complete."
    }

    Start-Sleep -Seconds 10
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

$taskResult = (Get-ScheduledTaskInfo -TaskName "$taskName").LastTaskResult
Write-Host "Install exit code: $taskResult"
if ($taskResult -ne 0) { $exitCode = $taskResult }

# Wait a moment for registry to update after installation
Start-Sleep -Seconds 2

Write-Host "Removing ScheduledTask: $taskName."
Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Remove-Item -Path $exeFilePath -Force -ErrorAction SilentlyContinue
}

Exit $exitCode
