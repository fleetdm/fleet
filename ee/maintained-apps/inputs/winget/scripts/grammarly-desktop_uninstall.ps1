# Grammarly registers per user, so its uninstall entry lives in a user hive
# rather than under HKLM, and the uninstaller has to run as that user for the
# entry to go away with the files. Only the logged-in user's hive is searched,
# since that is the account the uninstaller runs as. Uninstall.exe also
# relaunches itself from a temp copy and exits immediately, so removal is only
# done once the entry is gone.

$softwareName = "Grammarly for Windows"

$exitCode = 0
$taskName = $null

function Get-GrammarlyEntry {
    param([string]$Sid, [string]$DisplayName)
    $roots = @(
        "Registry::HKEY_USERS\$Sid\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
        "Registry::HKEY_USERS\$Sid\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
    )
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -eq $DisplayName) { return $key }
        }
    }
    return $null
}

try {

$explorer = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Select-Object -First 1
if (-not $explorer) {
    Write-Host "No logged-in user found. Grammarly is installed per user and cannot be removed now."
    Exit 1
}
$userName = ($explorer | Invoke-CimMethod -MethodName GetOwner).User
$userSid = ($explorer | Invoke-CimMethod -MethodName GetOwnerSid).Sid
if (-not $userName -or -not $userSid) {
    Write-Host "Could not resolve the logged-in user."
    Exit 1
}

$entry = Get-GrammarlyEntry -Sid $userSid -DisplayName $softwareName
if (-not $entry) {
    Write-Host "Uninstall entry not found for '$softwareName' under user $userName."
    Exit 1
}

$uninstallCommand = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }

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

$lastRun = (Get-ScheduledTaskInfo -TaskName "$taskName").LastRunTime
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

# The uninstaller can exit before the first poll, so treat an advanced
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
    if ((Get-Date) -ge $deadline) { Throw "Timed-out waiting for the uninstaller to complete." }
    Write-Host "Waiting for the uninstaller to complete..."
    Start-Sleep -Seconds 10
}

# The relaunched copy keeps working after the task exits, so wait for the
# uninstall entry to disappear before reporting success.
$deadline = (Get-Date).AddSeconds(180)
while (Get-GrammarlyEntry -Sid $userSid -DisplayName $softwareName) {
    if ((Get-Date) -ge $deadline) {
        Write-Host "'$softwareName' is still registered 180s after the uninstaller finished."
        $exitCode = 1
        break
    }
    Write-Host "Waiting for '$softwareName' to be removed..."
    Start-Sleep -Seconds 5
}

if ($exitCode -eq 0) { Write-Host "'$softwareName' removed." }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # An AtLogOn task left registered would run the uninstaller again at the
    # next logon, so remove it even when the wait above failed.
    if ($taskName -and (Get-ScheduledTask -TaskName "$taskName" -ErrorAction SilentlyContinue)) {
        Write-Host "Removing ScheduledTask: $taskName."
        Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false
    }
}

Exit $exitCode
