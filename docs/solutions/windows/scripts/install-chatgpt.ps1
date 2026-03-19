$taskName = "Install ChatGPT Desktop"

# Encode the script as Base64, so we can use it with a scheduled task
$encodedCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(@'
# These variables need to be inside the script block
$wingetPath = Get-Command winget.exe -ErrorAction SilentlyContinue
$appName = "ChatGPT Desktop"

Write-Output "Please do not close this window."

if (-not $wingetPath) {
    Write-Output "Installing WinGet in order to install $appName..."
    $filePath = Join-Path $env:TEMP "winget.msixbundle"
    Invoke-WebRequest -Uri https://aka.ms/getwinget -OutFile $filePath
    Add-AppxPackage $filePath
}

$args = @(
    "--id", "9NT1R1C2HH7J"
    "--source", "msstore"
    "--silent"
    "--accept-package-agreements"
    "--accept-source-agreements"
)

Write-Output "`nInstalling $appName...`n"
winget install $args
'@))


# Pop up at the top, shows a PowerShell window
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -EncodedCommand $encodedCommand"

# `EndBoundary` to automatically delete the task with `DeleteExpiredTaskAfter` below
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date)
$trigger.EndBoundary = (Get-Date).AddSeconds(5).ToString("s")

$currentUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName

# Use `-RunLevel Highest` here so that `Unregister-ScheduledTask` will work later; otherwise it fails with a `PermissionDenied` error
$principal = New-ScheduledTaskPrincipal -UserId $currentUser -RunLevel Highest

# `ExecutionTimeLimit` in case it hangs
$settings = New-ScheduledTaskSettingsSet -RunOnlyIfNetworkAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 10) -DeleteExpiredTaskAfter (New-TimeSpan -Seconds 5)

$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal -Settings $settings

# If a task already has this name, delete it first
if ((Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue)) {
    Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false
}

Write-Host "Logged in user is $currentUser."
Write-Host "Starting ScheduledTask."

# Register and start task
Register-ScheduledTask "$taskName" -InputObject $task
Start-ScheduledTask -TaskName "$taskName"
