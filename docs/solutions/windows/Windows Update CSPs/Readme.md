# **Windows Update CSP Profiles for Fleet**

Custom XML profiles for managing Windows Update behavior via Fleet MDM
using Microsoft\'s Policy CSP - Update nodes. These profiles were built
and tested on a Windows 11 24H2 VM enrolled in Fleet 4.83.

## **PREREQUISITES**

1.  Fleet 4.77.0 or later

2.  Environment variable
    > FLEET_MDM_ENABLE_CUSTOM_OS_UPDATES_AND_FILEVAULT=1 set on the
    > Fleet server (requires restart)

3.  Fleet\'s built-in Windows OS update enforcement **disabled** for any
    > team these profiles are applied to (Controls \> OS updates \>
    > remove any Windows deadline/grace period). The custom CSP profiles
    > take over update management and will conflict with Fleet\'s
    > built-in controls if both are active.

## **PROFILE OVERVIEW**

Seven XML files, each targeting a specific update behavior. Deploy
individually or combine into a single profile.

**01-auto-update-and-schedule.xml\
**Sets auto-update mode to \"install and restart at scheduled time\"
(AllowAutoUpdate=3), schedules installs for Saturdays at 2 AM.

-   AllowAutoUpdate = 3 (auto install + restart at scheduled time)

-   ScheduledInstallDay = 7 (Saturday)

-   ScheduledInstallTime = 2 (2:00 AM)

**02-defer-feature-and-quality-updates.xml\
**Postpones feature updates by 30 days and quality/security updates by 7
days from release.

-   DeferFeatureUpdatesPeriodInDays = 30 (max 365)

-   DeferQualityUpdatesPeriodInDays = 7 (max 30)

**03-active-hours.xml\
**Prevents restarts during business hours. Windows will not auto-restart
between 8 AM and 6 PM.

-   ActiveHoursStart = 8

-   ActiveHoursEnd = 18

**04-deadlines-and-grace-periods.xml\
**Forces update installation after a deadline, regardless of active
hours. Grace period gives recently powered-on devices (e.g., someone
returning from vacation) a buffer before forced restart.

-   ConfigureDeadlineForFeatureUpdates = 14 days

-   ConfigureDeadlineForQualityUpdates = 7 days

-   ConfigureDeadlineGracePeriod = 2 days

-   ConfigureDeadlineGracePeriodForFeatureUpdates = 2 days

**05-pin-to-windows-11-24h2.xml\
**Pins devices to Windows 11 24H2. Devices still receive
security/quality updates for 24H2 but will not be offered feature
updates to newer versions (e.g., 25H1).

-   ProductVersion = \"Windows 11\" (format: chr)

-   TargetReleaseVersion = \"24H2\" (format: chr)

Note: these two values use chr (string) format, not int. This is a
common mistake in examples online.

**06-exclude-drivers-and-disable-wu-access.xml\
**Excludes driver updates from quality updates and removes the user\'s
ability to manually scan/download/install from the Windows Update UI.

-   ExcludeWUDriversInQualityUpdate = 1

-   SetDisableUXWUAccess = 1

Considerations: disabling Windows Update access means end users cannot
manually check for updates. If that\'s too aggressive, remove the
SetDisableUXWUAccess setting and only deploy the driver exclusion.

**07-suppress-notifications-during-active-hours.xml\
**Suppresses Windows Update notifications during active hours, except
for restart warnings. Requires Windows 11 22H2 or later.

-   NoUpdateNotificationsDuringActiveHours = 1

## **HOW TO DEPLOY**

**Individual profiles:\
**Upload each XML file separately via Controls \> OS settings \> Custom
settings. This gives you granular control to enable/disable specific
behaviors per team.

**Combined profile:\
**Combine all \<Replace\> blocks from the individual files into a single
XML file. Upload as one profile if you want everything applied at once.

**Deployment methods:**

-   UI: Controls \> OS settings \> Custom settings \> Upload

-   GitOps: Add to controls.windows_settings.configuration_profiles in
    > team YAML

-   API: POST /api/v1/fleet/configuration_profiles with XML content

Profiles apply on next MDM check-in (typically within an hour). Force a
check-in from the host details page if needed.

## **HOW TO VERIFY**

All settings write to the registry at
HKLM:\\SOFTWARE\\Microsoft\\PolicyManager\\current\\device\\Update.

**Check all values at once (PowerShell, run on the device):**

****\$keys = @(

\'AllowAutoUpdate\',

\'ScheduledInstallDay\',

\'ScheduledInstallTime\',

\'DeferFeatureUpdatesPeriodInDays\',

\'DeferQualityUpdatesPeriodInDays\',

\'ActiveHoursStart\',

\'ActiveHoursEnd\',

\'ConfigureDeadlineForFeatureUpdates\',

\'ConfigureDeadlineForQualityUpdates\',

\'ConfigureDeadlineGracePeriod\',

\'ConfigureDeadlineGracePeriodForFeatureUpdates\',

\'ProductVersion\',

\'TargetReleaseVersion\',

\'ExcludeWUDriversInQualityUpdate\',

\'SetDisableUXWUAccess\',

\'NoUpdateNotificationsDuringActiveHours\'

)

\$regPath =
\'HKLM:\\SOFTWARE\\Microsoft\\PolicyManager\\current\\device\\Update\'

foreach (\$key in \$keys) {

\$val = (Get-ItemProperty -Path \$regPath -Name \$key -ErrorAction
SilentlyContinue).\$key

Write-Host \"\$key = \$val\"

}

**Expected values:**

  -----------------------------------------------------------------------
  **Key**                                                **Expected**
  ------------------------------------------------------ ----------------
  AllowAutoUpdate                                        3

  ScheduledInstallDay                                    7

  ScheduledInstallTime                                   2

  DeferFeatureUpdatesPeriodInDays                        30

  DeferQualityUpdatesPeriodInDays                        7

  ActiveHoursStart                                       8

  ActiveHoursEnd                                         18

  ConfigureDeadlineForFeatureUpdates                     14

  ConfigureDeadlineForQualityUpdates                     7

  ConfigureDeadlineGracePeriod                           2

  ConfigureDeadlineGracePeriodForFeatureUpdates          2

  ProductVersion                                         Windows 11

  TargetReleaseVersion                                   24H2

  ExcludeWUDriversInQualityUpdate                        1

  SetDisableUXWUAccess                                   1

  NoUpdateNotificationsDuringActiveHours                 1
  -----------------------------------------------------------------------

**Per-profile verification (if deploying individually):**

-   01: Check AllowAutoUpdate=3, ScheduledInstallDay=7,
    > ScheduledInstallTime=2

-   02: Check DeferFeatureUpdatesPeriodInDays=30,
    > DeferQualityUpdatesPeriodInDays=7

-   03: Check ActiveHoursStart=8, ActiveHoursEnd=18

-   04: Check all four deadline/grace period values

-   05: Check ProductVersion=\"Windows 11\",
    > TargetReleaseVersion=\"24H2\"

-   06: Check ExcludeWUDriversInQualityUpdate=1, SetDisableUXWUAccess=1

-   07: Check NoUpdateNotificationsDuringActiveHours=1

You can also verify profile delivery in Fleet on the host details page,
or on the device via Settings \> Accounts \> Access work or school \>
Fleet connection \> Info \> Create Report.

## **CUSTOMIZATION**

All values in these profiles are examples. Adjust to match your
environment:

-   **Schedule:** Change ScheduledInstallDay (0=every day, 1=Sun through
    > 7=Sat) and ScheduledInstallTime (0-23, 24hr format) to match your
    > maintenance window

-   **Deferral periods:** DeferFeatureUpdatesPeriodInDays supports up
    > to 365. DeferQualityUpdatesPeriodInDays supports up to 30.

-   **Deadlines:** Setting a deadline to 0 means \"install
    > immediately.\" Higher values give users more time to voluntarily
    > restart.

-   **Version pin:** Change TargetReleaseVersion to whichever version
    > you want to pin to (e.g., \"23H2\", \"24H2\")

-   **Active hours:** ActiveHoursStart and ActiveHoursEnd accept 0-23.
    > Max range between them is 18 hours.

## **KNOWN LIMITATIONS**

1.  **No per-KB blocking.** There is no CSP node to block individual
    > Windows updates by KB ID. If you need that level of control, you
    > need WSUS or a WSUS-like solution.

2.  **No custom update notifications.** Nudge is macOS-only. Windows
    > Update notifications cannot be customized (branded, reworded,
    > etc.) via MDM. You can suppress them or allow them, but not change
    > their content. A PowerShell-based toast notification script is
    > available as a workaround (see nudge-toast-notification.ps1).

3.  **NoUpdateNotificationsDuringActiveHours requires Windows 11
    > 22H2+.** Older builds will ignore this setting.

4.  **SetDisableUXWUAccess is aggressive.** Users lose the ability to
    > manually trigger updates. Consider whether this is appropriate for
    > your environment before deploying.

## **REFERENCES**

-   [[Microsoft Policy CSP -
    > Update]{.underline}](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update)

-   [[Fleet: Custom OS
    > Settings]{.underline}](https://fleetdm.com/guides/custom-os-settings)

-   [[Fleet: Enforce OS
    > Updates]{.underline}](https://fleetdm.com/guides/enforce-os-updates)
