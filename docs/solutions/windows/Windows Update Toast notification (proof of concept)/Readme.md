# **Windows Update Toast Notifications for Fleet (PoC)**

## **What This Is**

A proof-of-concept \"Nudge for Windows\" implementation. Two PowerShell
scripts that display native Windows toast notifications to remind users
to install pending updates and restart.

No third-party dependencies. Uses the built-in .NET
Windows.UI.Notifications API that ships with Windows 10 and Windows 11.

## **Why It\'s Needed**

Fleet scripts run as SYSTEM. Toast notifications require a user session
to display. The workaround: Fleet deploys a script that creates a
Scheduled Task running as the logged-in user. The scheduled task runs
the actual toast notification script in user context.

## **Files**

  ------------------------------------------------------------------------------------
  **Script**                 **Purpose**                                   **Run
                                                                           Context**
  -------------------------- --------------------------------------------- -----------
  install-toast-task.ps1     Deployment script. Embeds the toast script    SYSTEM
                             inline, writes it to disk, creates a          (Fleet)
                             Scheduled Task as the logged-in user.         
                             Self-contained, single-file. Deploy via       
                             Fleet.                                        

  uninstall-toast-task.ps1   Cleanup script. Removes the scheduled task,   SYSTEM
                             deletes deployed files, clears toast history  (Fleet)
                             from Action Center. Idempotent.               

  show-update-toast.ps1      Standalone toast notification script for      User
                             reference/testing. Same code that gets        
                             embedded into the install script. Not needed  
                             for deployment.                               
  ------------------------------------------------------------------------------------

## **How It Works**

1.  Fleet runs install-toast-task.ps1 as SYSTEM

2.  Script detects the logged-in user via explorer.exe process owner
    > (fallback: query user)

3.  Writes the toast notification script to
    > C:\\ProgramData\\Fleet\\ToastNotification\\show-update-toast.ps1

4.  Creates a Scheduled Task named FleetWindowsUpdateToast that:

    -   Runs as the logged-in user (interactive, non-elevated)

    -   Triggers on a repeating interval (configurable) and at logon

    -   Executes the toast script with -WindowStyle Hidden

5.  Toast script checks for pending updates via Microsoft.Update.Session
    > COM object

6.  If updates are pending (or a reboot is required), displays a toast
    > notification with \"Update Now\" and \"Remind Me Later\" buttons

7.  \"Update Now\" opens ms-settings:windowsupdate directly

If no updates are pending and no reboot is required, the script exits
silently.

## **Configuration**

All configuration is in the CONFIGURATION section at the top of
install-toast-task.ps1.

### **Install Script Settings**

  -------------------------------------------------------------------------------
  **Variable**   **Default**   **Description**
  -------------- ------------- --------------------------------------------------
  \$TriggerNow   \$true        Fire the notification immediately after install
                               (for testing)

  \$ForceShow    \$true        Skip the pending update check and always show the
                               notification (for testing). Set to \$false for
                               production.
  -------------------------------------------------------------------------------

### **Toast Notification Settings (embedded in install script)**

  -----------------------------------------------------------------------------------------
  **Variable**          **Default**                                **Description**
  --------------------- ------------------------------------------ ------------------------
  \$CompanyName         \"IT Department\"                          Attribution text shown
                                                                   below the notification
                                                                   body

  \$HeroTitle           \"Windows Update Available\"               Notification title.
                                                                   Changes to \"Restart
                                                                   Required\" if a reboot
                                                                   is pending.

  \$HeroMessage         (pending updates message)                  Notification body.
                                                                   Auto-adjusts based on
                                                                   update count and reboot
                                                                   state.

  \$ActionButtonText    \"Update Now\"                             Primary button label

  \$DismissButtonText   \"Remind Me Later\"                        Dismiss button label

  \$LogoPath            C:\\ProgramData\\Fleet\\company-logo.png   Path to a company logo
                                                                   (PNG). If the file
                                                                   exists, it shows as a
                                                                   circular icon on the
                                                                   toast. Optional.
  -----------------------------------------------------------------------------------------

### **Schedule**

The current build uses a 5-minute repeat interval for testing. For
production, swap the trigger in the install script:

**Testing (current):**

****\$triggerRepeat = New-ScheduledTaskTrigger -Once -At (Get-Date) \`

-RepetitionInterval (New-TimeSpan -Minutes 5) \`

-RepetitionDuration (New-TimeSpan -Days 1)

**Production:**

****\$triggerDaily = New-ScheduledTaskTrigger -Daily -At \"10:00\"



## **Deployment via Fleet**

### **Install**

1.  Upload install-toast-task.ps1 to Fleet: Controls \> Scripts \> Add
    > script

2.  Run it against the target hosts

3.  Verify the scheduled task exists on the device:

4.  Get-ScheduledTask -TaskName \"FleetWindowsUpdateToast\" \|
    > Format-List

### **Uninstall**

1.  Upload uninstall-toast-task.ps1 to Fleet: Controls \> Scripts \> Add
    > script

2.  Run it against the target hosts

3.  Removes the scheduled task, deployed script files, and clears toast
    > history

### **Custom Branding**

To add a company logo, deploy a PNG file to
C:\\ProgramData\\Fleet\\company-logo.png on target devices before
running the install script. The toast notification will display it as a
circular icon overlay.

## **Verification**

On a target device, run in PowerShell:

# Check the scheduled task exists and is ready

Get-ScheduledTask -TaskName \"FleetWindowsUpdateToast\" \| Format-List
TaskName, State, Description

\# Check the deployed script exists

Test-Path
\"C:\\ProgramData\\Fleet\\ToastNotification\\show-update-toast.ps1\"

\# Manually trigger the task

Start-ScheduledTask -TaskName \"FleetWindowsUpdateToast\"

\# Check last run result (0 = success)

Get-ScheduledTaskInfo -TaskName \"FleetWindowsUpdateToast\" \|
Select-Object LastRunTime, LastTaskResult



## **Limitations and Considerations**

-   **App identity:** The toast appears as \"Windows PowerShell\"
    > because it uses PowerShell\'s built-in AUMID. Custom app
    > names/icons in the toast header require registering a custom AppId
    > in the registry or using a compiled helper. Doable but heavier.

-   **User detection:** If no interactive user is logged in when the
    > install script runs, it will exit with an error. The scheduled
    > task only gets created for the user who was logged in at install
    > time. Multi-user devices would need additional handling.

-   **Focus Assist / DND:** Windows Focus Assist can suppress toast
    > notifications. The scenario=\"reminder\" attribute helps
    > (reminders break through some Focus Assist modes) but is not
    > guaranteed.

-   **Notification expiry:** Toasts expire after 8 hours. If the user
    > does not interact with it, it disappears from Action Center.

-   **Fleet script context:** Fleet currently runs all scripts as SYSTEM
    > (see
    > [[fleetdm/fleet#32587]{.underline}](https://github.com/fleetdm/fleet/issues/32587)).
    > The scheduled task workaround is necessary until Fleet supports
    > Contextual script execution.

## **References**

-   [[Windows.UI.Notifications API (Microsoft
    > Docs)]{.underline}](https://learn.microsoft.com/en-us/uwp/api/windows.ui.notifications)

-   [[Toast notification XML
    > schema]{.underline}](https://learn.microsoft.com/en-us/windows/apps/design/shell/tiles-and-notifications/toast-xml-schema)

-   [[Fleet:
    > Scripts]{.underline}](https://fleetdm.com/docs/using-fleet/scripts)

-   [[Fleet issue #32587: Contextual script
    > execution]{.underline}](https://github.com/fleetdm/fleet/issues/32587)
