# Grammarly registers per user, so its uninstall entry lives in the user's own
# hive rather than under HKLM. This reads the uninstaller path out of the loaded
# user hives, then runs it through a scheduled task in the logged-in user's
# session so the user's registry entry is removed along with the files.

$softwareName = "Grammarly for Windows"

$exitCode = 0

try {

$userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Invoke-CimMethod -MethodName getowner).User
if (-not $userName) {
    Write-Host "No logged-in user found. Grammarly is installed per user and cannot be removed now."
    Exit 1
}

$roots = [System.Collections.Generic.List[string]]::new()
foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.Name -match '_Classes$') { continue }
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
}

$uninstallCommand = $null
foreach ($root in $roots) {
    foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
        $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
        if ($key.DisplayName -ne $softwareName) { continue }
        $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
        break
    }
    if ($uninstallCommand) { break }
}

if (-not $uninstallCommand) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

# Defensive parser for the three UninstallString shapes:
#   "C:\Users\me\AppData\Local\Grammarly\...\Uninstall.exe" /S -> quoted
#   C:\path with spaces\Uninstall.exe /S                       -> unquoted with spaces
#   MsiExec.exe /X{GUID}                                       -> bare token
$uninstallPath = $null
if ($uninstallCommand -match '^\s*"([^"]+)"') {
    $uninstallPath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
    $uninstallPath = $matches[1]
} elseif ($uninstallCommand -match '^\s*(\S+)') {
    $uninstallPath = $matches[1]
}

if (-not $uninstallPath -or -not (Test-Path $uninstallPath)) {
    Write-Host "Uninstaller not found at: $uninstallPath"
    Exit 1
}

Write-Host "Uninstall command: $uninstallPath /S"

# /S is the NSIS silent switch
$action = New-ScheduledTaskAction -Execute "$uninstallPath" -Argument "/S"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
$task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings

$taskName = "fleet-uninstall-grammarly"
Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

$startDate = Get-Date
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

$state = (Get-ScheduledTask -TaskName "$taskName").State
while ($state -ne "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting to run uninstaller..."

    if ((New-Timespan -Start $startDate -End (Get-Date)).TotalSeconds -gt 120) {
        Throw "Timed-out waiting for scheduled task state."
    }

    Start-Sleep -Seconds 1
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

$state = (Get-ScheduledTask -TaskName "$taskName").State
while ($state -eq "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting for uninstaller to complete..."

    if ((New-Timespan -Start $startDate -End (Get-Date)).TotalSeconds -gt 600) {
        Throw "Timed-out waiting for scheduled task to complete."
    }

    Start-Sleep -Seconds 10
    $state = (Get-ScheduledTask -TaskName "$taskName").State
}

$taskResult = (Get-ScheduledTaskInfo -TaskName "$taskName").LastTaskResult
Write-Host "Uninstall exit code: $taskResult"
if ($taskResult -ne 0) { $exitCode = $taskResult }

Start-Sleep -Seconds 2

Write-Host "Removing ScheduledTask: $taskName."
Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
