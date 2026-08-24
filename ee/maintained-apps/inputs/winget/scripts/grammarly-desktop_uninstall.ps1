# Grammarly registers per user, so its uninstall entry lives in a user hive
# rather than under HKLM, and the uninstaller has to run as that user for the
# entry to go away with the files. Uninstall.exe also relaunches itself from a
# temp copy and exits immediately, so removal is only done once the entry is gone.

$softwareName = "Grammarly for Windows"

$exitCode = 0

function Get-GrammarlyEntry {
    param([string]$DisplayName)
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots = @(
            "Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
            "Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
        )
        foreach ($root in $roots) {
            foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
                $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
                if ($key.DisplayName -eq $DisplayName) { return $key }
            }
        }
    }
    return $null
}

try {

$userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Invoke-CimMethod -MethodName getowner).User
if (-not $userName) {
    Write-Host "No logged-in user found. Grammarly is installed per user and cannot be removed now."
    Exit 1
}

$entry = Get-GrammarlyEntry -DisplayName $softwareName
if (-not $entry) {
    Write-Host "Uninstall entry not found for '$softwareName'."
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

Write-Host "Removing ScheduledTask: $taskName."
Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

# The relaunched copy keeps working after the task exits, so wait for the
# uninstall entry to disappear before reporting success.
$removeDate = Get-Date
while (Get-GrammarlyEntry -DisplayName $softwareName) {
    if ((New-Timespan -Start $removeDate -End (Get-Date)).TotalSeconds -gt 180) {
        Write-Host "'$softwareName' is still registered 180s after the uninstaller finished."
        Exit 1
    }
    Write-Host "Waiting for '$softwareName' to be removed..."
    Start-Sleep -Seconds 5
}

Write-Host "'$softwareName' removed."

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
