<#
.SYNOPSIS
    Windows Update Nudge Toast Notification
    Deploy as a Scheduled Task via the companion install script.

.DESCRIPTION
    Displays a native Windows toast notification reminding the user to install
    pending Windows updates. Uses the built-in .NET Windows.UI.Notifications API.
    Zero external dependencies.

    MUST run in user context (not SYSTEM) for the toast to appear.

.NOTES
    - Works on Windows 10 1809+ and Windows 11
    - Customize the variables in the CONFIGURATION section below
#>

# ============================================================
# CONFIGURATION - Customize these values
# ============================================================

$CompanyName       = "IT Department"               # e.g. "EasyGo IT"
$HeroTitle         = "Windows Update Available"
$HeroMessage       = "Your device has pending security updates. Please restart to apply them and keep your device protected."
$ActionButtonText  = "Update Now"
$DismissButtonText = "Remind Me Later"

# Path to a company logo (32x32 or 48x48 PNG). Optional.
# If the file doesn't exist, the notification uses the default app icon.
$LogoPath          = "$env:ProgramData\Fleet\company-logo.png"

# Set to $false to always show the notification (useful for testing)
$CheckForUpdates   = $true

# ============================================================
# CHECK FOR PENDING UPDATES
# ============================================================

if ($CheckForUpdates) {
    try {
        $UpdateSession  = New-Object -ComObject Microsoft.Update.Session
        $UpdateSearcher = $UpdateSession.CreateUpdateSearcher()
        $SearchResult   = $UpdateSearcher.Search("IsInstalled=0 AND IsHidden=0")
        $PendingCount   = $SearchResult.Updates.Count

        if ($PendingCount -eq 0) {
            # Also check for pending reboot before bailing out
            $rebootKeys = @(
                "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired",
                "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending"
            )
            $RebootPending = $false
            foreach ($key in $rebootKeys) {
                if (Test-Path $key) { $RebootPending = $true; break }
            }
            if (-not $RebootPending) {
                Write-Host "No pending updates or reboots. Skipping notification."
                exit 0
            }
        }
        else {
            $HeroMessage = "Your device has $PendingCount pending update(s). Please restart to apply them and keep your device protected."
            Write-Host "Found $PendingCount pending update(s)."
        }
    }
    catch {
        Write-Host "Could not check for updates: $_. Showing notification anyway."
    }
}

# ============================================================
# CHECK FOR PENDING REBOOT
# ============================================================

$RebootRequired = $false
$rebootKeys = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired",
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending"
)
foreach ($key in $rebootKeys) {
    if (Test-Path $key) { $RebootRequired = $true; break }
}

if ($RebootRequired) {
    $HeroTitle   = "Restart Required"
    $HeroMessage = "Your device needs to restart to finish installing security updates. Please save your work and restart soon."
}

# ============================================================
# BUILD AND DISPLAY TOAST NOTIFICATION
# ============================================================

[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$AppId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'

$imageXml = ""
if ($LogoPath -and (Test-Path $LogoPath)) {
    $imageXml = "<image placement='appLogoOverride' hint-crop='circle' src='file:///$($LogoPath.Replace('\','/'))'/>"
}

$ToastXml = @"
<toast duration="long" scenario="reminder">
    <visual>
        <binding template="ToastGeneric">
            $imageXml
            <text>$HeroTitle</text>
            <text>$HeroMessage</text>
            <text placement="attribution">$CompanyName</text>
        </binding>
    </visual>
    <actions>
        <action content="$ActionButtonText" arguments="ms-settings:windowsupdate" activationType="protocol"/>
        <action content="$DismissButtonText" arguments="dismiss" activationType="system"/>
    </actions>
    <audio src="ms-winsoundevent:Notification.Reminder"/>
</toast>
"@

try {
    $XmlDoc = [Windows.Data.Xml.Dom.XmlDocument]::new()
    $XmlDoc.LoadXml($ToastXml)
    $Toast = [Windows.UI.Notifications.ToastNotification]::new($XmlDoc)
    $Toast.ExpirationTime = [DateTimeOffset]::Now.AddHours(8)
    $Notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($AppId)
    $Notifier.Show($Toast)
    Write-Host "Toast notification displayed."
    Write-Host "  Title: $HeroTitle"
    Write-Host "  Reboot pending: $RebootRequired"
    exit 0
}
catch {
    Write-Host "ERROR: Failed to display toast notification: $_"
    exit 1
}
