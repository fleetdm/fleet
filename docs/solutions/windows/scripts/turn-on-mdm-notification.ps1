# Encode the script as Base64, so we can use it with a scheduled task
$encodedCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(@'
# Locate the MDM Enrollment Key in the registry
$enrollmentKey = Get-Item -Path HKLM:\SOFTWARE\Microsoft\Enrollments\* | Get-ItemProperty | Where-Object {$_.ProviderID -eq 'Fleet'} | Where-Object {$_.EnrollmentState -match '1|6|13'}

if($enrollmentKey){
    $isMDMTurnedOn = $true
} else {
    $isMDMTurnedOn = $false
}

# Set the task name now, so we can remove it
$taskName = "Turn on MDM notification"

if ($isMDMTurnedOn) {
    Write-Output "Thank you for turning MDM on."
    Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false
    Start-Sleep -Seconds 10
} else {
    $title = "Migrate to Fleet"
    $message = "Mobile device management is off. MDM allows your organization to change settings and install software.
    
Turn on MDM by following these steps:
    
Close this window, go to Settings and search `"Access work or school`".
    
Select **Connect** and enter your work email and password.
    
Open Fleet Desktop (Fleet icon) in your system tray (^) and select **Refetch** on your **My device** page to tell your organization that MDM is on.
    
This **Migrate to Fleet** window will pop up every 5 minutes until you finish."
    
# Send the message
(New-Object -ComObject WScript.Shell).Popup($message, 0, $title, 0)
}
'@))

# Pop up at the top, shows a PowerShell window
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -EncodedCommand $encodedCommand"

$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 5)
$currentUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName

# Use `-RunLevel Highest` here so that `Unregister-ScheduledTask` will work later; otherwise it fails with a `PermissionDenied` error
$principal = New-ScheduledTaskPrincipal -UserId "$currentUser" -RunLevel Highest

# `ExecutionTimeLimit` is used in case the user didn't close the popup, so that it will take focus again
$settings = New-ScheduledTaskSettingsSet -RunOnlyIfNetworkAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 4)

$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal -Settings $settings

Write-Host "Logged in user is $currentUser."
Write-Host "Starting ScheduledTask."

# Register and start task
$taskName = "Turn on MDM notification"
Register-ScheduledTask "$taskName" -InputObject $task
Start-ScheduledTask -TaskName "$taskName"
