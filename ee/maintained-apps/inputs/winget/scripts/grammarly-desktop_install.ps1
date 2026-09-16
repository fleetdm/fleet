# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Grammarly installs per user into %LOCALAPPDATA%, so it has to run as the
# logged-in user rather than as SYSTEM; this runs the installer through a
# scheduled task in that user's session. Nobody logged in means no install.

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0
$taskName = $null
$publicCopy = $null

try {

$explorer = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Select-Object -First 1
$userName = if ($explorer) { ($explorer | Invoke-CimMethod -MethodName GetOwner).User } else { $null }
if (-not $userName) {
    Write-Host "No logged-in user found. Grammarly installs per user and cannot be installed now."
    Exit 1
}

# Copy the installer to a public folder so that the logged-in user can read it
$exeFilename = Split-Path $exeFilePath -leaf
Copy-Item -Path $exeFilePath -Destination "${env:PUBLIC}" -Force
$publicCopy = "${env:PUBLIC}\$exeFilename"

# /S is the NSIS silent switch
$action = New-ScheduledTaskAction -Execute "$publicCopy" -Argument "/S"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
$task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings

$taskName = "fleet-install-$exeFilename"
Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

$lastRun = (Get-ScheduledTaskInfo -TaskName "$taskName").LastRunTime
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

# A short install can finish before the first poll, so treat an advanced
# LastRunTime as proof it ran rather than waiting to observe "Running".
$started = $false
$deadline = (Get-Date).AddSeconds(120)
while ((Get-Date) -lt $deadline) {
    $info = Get-ScheduledTaskInfo -TaskName "$taskName"
    if ((Get-ScheduledTask -TaskName "$taskName").State -eq "Running" -or $info.LastRunTime -ne $lastRun) {
        $started = $true
        break
    }
    Start-Sleep -Seconds 1
}
if (-not $started) { Throw "Scheduled task never started." }

$deadline = (Get-Date).AddSeconds(600)
while ((Get-ScheduledTask -TaskName "$taskName").State -eq "Running") {
    if ((Get-Date) -ge $deadline) { Throw "Timed-out waiting for the installer to complete." }
    Write-Host "Waiting for the installer to complete..."
    Start-Sleep -Seconds 10
}

$taskResult = (Get-ScheduledTaskInfo -TaskName "$taskName").LastTaskResult
Write-Host "Install exit code: $taskResult"
if ($taskResult -ne 0) { $exitCode = $taskResult }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # An AtLogOn task left registered would run the installer again at the next
    # logon, so remove it even when the wait above failed.
    if ($taskName -and (Get-ScheduledTask -TaskName "$taskName" -ErrorAction SilentlyContinue)) {
        Write-Host "Removing ScheduledTask: $taskName."
        Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false
    }
    if ($publicCopy) {
        Remove-Item -Path $publicCopy -Force -ErrorAction SilentlyContinue
    }
}

Exit $exitCode
