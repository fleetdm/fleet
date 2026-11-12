<#
.SYNOPSIS
    Fleet Deployment Script: Install Windows Update Toast Notification Scheduled Task
    Deploy via Fleet: Controls > Scripts > Add script

.DESCRIPTION
    This is a SINGLE-FILE deployment script. The toast notification script is
    embedded inline below so Fleet can deploy it without needing a companion file.

    This script runs as SYSTEM (Fleet's default) and:
    1. Writes the embedded toast notification script to a local directory
    2. Creates a Scheduled Task that runs as the logged-in user
    3. Optionally triggers the task immediately for testing

    The scheduled task runs daily at 10:00 AM and at user logon.
    The toast script checks for pending updates and only shows the
    notification if updates are available (unless ForceShow is enabled).

.NOTES
    - Run this script via Fleet to deploy the toast notification system
    - To uninstall, deploy the companion uninstall-toast-task.ps1
    - To test without real updates, set $ForceShow = $true below
#>

# ============================================================
# CONFIGURATION
# ============================================================

$TaskName        = "FleetWindowsUpdateToast"
$TaskDescription = "Displays a toast notification when Windows updates are pending."
$ScriptDir       = "$env:ProgramData\Fleet\ToastNotification"
$ScriptName      = "show-update-toast.ps1"
$ScriptPath      = Join-Path $ScriptDir $ScriptName

# Schedule: every 5 minutes for testing
# For production, switch the trigger block below to: New-ScheduledTaskTrigger -Daily -At "10:00"

# Set to $true to fire the notification immediately after install (for testing)
$TriggerNow      = $true

# Set to $true to force the toast to show even with no pending updates (for testing)
$ForceShow       = $true

# ============================================================
# FIND LOGGED-IN USER
# ============================================================

$LoggedInUser = $null
try {
    $explorerProc = Get-Process -Name explorer -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($explorerProc) {
        $owner = (Get-CimInstance Win32_Process -Filter "ProcessId = $($explorerProc.Id)").GetOwner()
        if ($owner.Domain -and $owner.User) {
            $LoggedInUser = "$($owner.Domain)\$($owner.User)"
        }
    }
}
catch {
    Write-Host "Could not determine logged-in user via explorer process: $_"
}

# Fallback: query session
if (-not $LoggedInUser) {
    try {
        $session = query user 2>$null | Where-Object { $_ -match 'Active' } | Select-Object -First 1
        if ($session -match '^\s*>?(\S+)') {
            $username = $Matches[1]
            $LoggedInUser = "$env:COMPUTERNAME\$username"
        }
    }
    catch {
        Write-Host "Could not determine logged-in user via query user: $_"
    }
}

if (-not $LoggedInUser) {
    Write-Host "ERROR: No logged-in user found. Cannot create user-context scheduled task."
    Write-Host "The toast notification requires a logged-in user to display."
    exit 1
}

Write-Host "Logged-in user: $LoggedInUser"

# ============================================================
# DEPLOY TOAST SCRIPT (embedded inline)
# ============================================================

if (-not (Test-Path $ScriptDir)) {
    New-Item -Path $ScriptDir -ItemType Directory -Force | Out-Null
    Write-Host "Created directory: $ScriptDir"
}

# Determine CheckForUpdates value based on ForceShow setting
$checkForUpdatesValue = if ($ForceShow) { '$false' } else { '$true' }

$toastScriptContent = @"
<#
.SYNOPSIS
    Windows Update Nudge Toast Notification
    Deployed by Fleet via scheduled task.

.DESCRIPTION
    Displays a native Windows toast notification reminding the user to install
    pending Windows updates. Uses the built-in .NET Windows.UI.Notifications API.
    Zero external dependencies. MUST run in user context (not SYSTEM).

.NOTES
    - Works on Windows 10 1809+ and Windows 11
    - Customize the variables in the CONFIGURATION section below
#>

# ============================================================
# CONFIGURATION
# ============================================================

`$CompanyName       = "IT Department"
`$HeroTitle         = "Windows Update Available"
`$HeroMessage       = "Your device has pending security updates. Please restart to apply them and keep your device protected."
`$ActionButtonText  = "Update Now"
`$DismissButtonText = "Remind Me Later"
`$LogoPath          = "`$env:ProgramData\Fleet\company-logo.png"
`$CheckForUpdates   = $checkForUpdatesValue

# ============================================================
# CHECK FOR PENDING UPDATES
# ============================================================

if (`$CheckForUpdates) {
    try {
        `$UpdateSession  = New-Object -ComObject Microsoft.Update.Session
        `$UpdateSearcher = `$UpdateSession.CreateUpdateSearcher()
        `$SearchResult   = `$UpdateSearcher.Search("IsInstalled=0 AND IsHidden=0")
        `$PendingCount   = `$SearchResult.Updates.Count

        if (`$PendingCount -eq 0) {
            `$rebootKeys = @(
                "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired",
                "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending"
            )
            `$RebootPending = `$false
            foreach (`$key in `$rebootKeys) {
                if (Test-Path `$key) { `$RebootPending = `$true; break }
            }
            if (-not `$RebootPending) {
                Write-Host "No pending updates or reboots. Skipping notification."
                exit 0
            }
        }
        else {
            `$HeroMessage = "Your device has `$PendingCount pending update(s). Please restart to apply them and keep your device protected."
            Write-Host "Found `$PendingCount pending update(s)."
        }
    }
    catch {
        Write-Host "Could not check for updates: `$_. Showing notification anyway."
    }
}

# ============================================================
# CHECK FOR PENDING REBOOT
# ============================================================

`$RebootRequired = `$false
`$rebootKeys = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired",
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending"
)
foreach (`$key in `$rebootKeys) {
    if (Test-Path `$key) { `$RebootRequired = `$true; break }
}

if (`$RebootRequired) {
    `$HeroTitle   = "Restart Required"
    `$HeroMessage = "Your device needs to restart to finish installing security updates. Please save your work and restart soon."
}

# ============================================================
# BUILD AND DISPLAY TOAST NOTIFICATION
# ============================================================

[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

`$AppId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'

`$imageXml = ""
if (`$LogoPath -and (Test-Path `$LogoPath)) {
    `$imageXml = "<image placement='appLogoOverride' hint-crop='circle' src='file:///`$(`$LogoPath.Replace('\','/'))'/>"
}

`$ToastXml = @'
<toast duration="long" scenario="reminder">
    <visual>
        <binding template="ToastGeneric">
            {0}
            <text>{1}</text>
            <text>{2}</text>
            <text placement="attribution">{3}</text>
        </binding>
    </visual>
    <actions>
        <action content="{4}" arguments="ms-settings:windowsupdate" activationType="protocol"/>
        <action content="{5}" arguments="dismiss" activationType="system"/>
    </actions>
    <audio src="ms-winsoundevent:Notification.Reminder"/>
</toast>
'@ -f `$imageXml, `$HeroTitle, `$HeroMessage, `$CompanyName, `$ActionButtonText, `$DismissButtonText

try {
    `$XmlDoc = [Windows.Data.Xml.Dom.XmlDocument]::new()
    `$XmlDoc.LoadXml(`$ToastXml)
    `$Toast = [Windows.UI.Notifications.ToastNotification]::new(`$XmlDoc)
    `$Toast.ExpirationTime = [DateTimeOffset]::Now.AddHours(8)
    `$Notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier(`$AppId)
    `$Notifier.Show(`$Toast)
    Write-Host "Toast notification displayed."
    Write-Host "  Title: `$HeroTitle"
    Write-Host "  Reboot pending: `$RebootRequired"
    exit 0
}
catch {
    Write-Host "ERROR: Failed to display toast notification: `$_"
    exit 1
}
"@

Set-Content -Path $ScriptPath -Value $toastScriptContent -Force
Write-Host "Deployed toast script to: $ScriptPath"

# ============================================================
# CREATE SCHEDULED TASK
# ============================================================

# Remove existing task if present
$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    Write-Host "Removed existing scheduled task: $TaskName"
}

# Action: run powershell with the toast script, hidden window
$action = New-ScheduledTaskAction `
    -Execute "powershell.exe" `
    -Argument "-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$ScriptPath`""

# Trigger: starts now, repeats every 5 minutes (testing mode)
# For production, swap this for: New-ScheduledTaskTrigger -Daily -At "10:00"
$triggerRepeat = New-ScheduledTaskTrigger -Once -At (Get-Date) `
    -RepetitionInterval (New-TimeSpan -Minutes 5) `
    -RepetitionDuration (New-TimeSpan -Days 1)

$triggerLogon = New-ScheduledTaskTrigger -AtLogOn -User $LoggedInUser

# Principal: run as the logged-in user (interactive, not elevated)
$principal = New-ScheduledTaskPrincipal `
    -UserId $LoggedInUser `
    -LogonType Interactive `
    -RunLevel Limited

# Settings
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -MultipleInstances IgnoreNew

# Register the task
Register-ScheduledTask `
    -TaskName $TaskName `
    -Description $TaskDescription `
    -Action $action `
    -Trigger @($triggerRepeat, $triggerLogon) `
    -Principal $principal `
    -Settings $settings `
    -Force | Out-Null

Write-Host "Scheduled task '$TaskName' created successfully."
Write-Host "  Runs as: $LoggedInUser"
Write-Host "  Schedule: Every 5 minutes (testing mode) + at logon"

# ============================================================
# TRIGGER IMMEDIATELY (if enabled)
# ============================================================

if ($TriggerNow) {
    Write-Host "Triggering toast notification now..."
    Start-ScheduledTask -TaskName $TaskName
    Start-Sleep -Seconds 2
    $taskInfo = Get-ScheduledTaskInfo -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($taskInfo -and $taskInfo.LastTaskResult -eq 0) {
        Write-Host "Toast notification triggered successfully."
    }
    else {
        Write-Host "Task triggered. Check the device for the notification."
        Write-Host "  Last result: $($taskInfo.LastTaskResult)"
    }
}

Write-Host ""
Write-Host "Done. To uninstall, run uninstall-toast-task.ps1 via Fleet."
exit 0
