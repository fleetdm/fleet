# Enforce OS updates

_Available in Fleet Premium_

In Fleet, you can enforce OS updates on your macOS, Windows, iOS, and iPadOS hosts remotely using the Fleet UI, Fleet API, or Fleet's GitOps workflow.

## Turning on enforcement

For Apple (macOS, iOS, and iPadOS) hosts, Apple provides a [list of available OS versions](https://gdmf.apple.com/v2/pmv) in the Apple Software Lookup Service. The update will only be enforced if you use a version in that list.

### Fleet UI

1. Head to the **Controls** > **OS updates** tab.

2. To enforce OS updates for macOS, iOS, or iPadOS, select the platform and set a **Minimum version** and **Deadline**.

3. For Windows, select **Windows** and set a **Deadline** and **Grace period**.

### Fleet API

Use the [modify team endpoint](https://fleetdm.com/docs/rest-api/rest-api#modify-team) to turn on minimum OS version enforcement. The relevant payload keys in the `mdm` object are:
+ `macos_updates`
+ `ios_updates`
+ `ipados_updates`
+ `windows_updates`

### GitOps

OS version enforcement options are declared within the [controls](https://fleetdm.com/docs/configuration/yaml-files#controls) section of a Fleet GitOps YAML file, using the following keys: 
+ [macos_updates](https://fleetdm.com/docs/configuration/yaml-files#macos-updates)
+ [ios_updates](https://fleetdm.com/docs/configuration/yaml-files#ios-updates)
+ [ipados_updates](https://fleetdm.com/docs/configuration/yaml-files#ipados-updates)
+ [windows_updates](https://fleetdm.com/docs/configuration/yaml-files#windows-updates)

## End user experience

### Apple (macOS, iOS, and iPadOS)

On macOS hosts, when a minimum version is enforced, end users see a native macOS notification (DDM) once per day. Users can choose to update ahead of the deadline or schedule it for that night. 24 hours before the deadline, the notification appears hourly and ignores Do Not Disturb. One hour before the deadline, the notification appears every 30 minutes and then every 10 minutes.

> Certain user preferences may suppress macOS update notifications. To prevent users from being surprised by a forced update or unexpected restart, consider communicating OS update deadlines through additional channels.

On iOS and iPadOS hosts, end users will see a notification in their Notification Center after the deadline. They can’t use their iPhone or iPad until the OS update is installed.

If the host was turned off when the deadline passed, the update will be scheduled an hour after it’s turned on.

If you set a past date (ex. yesterday) as the deadline, the end user will immediately be prompted to install the update. If they don't, the update will automatically install in one hour. Similarly, if you set the deadline to today, end users will experience the same behavior if it's after 12 PM (end user local time).

For hosts that use Automated Device Enrollment (ADE), if the device is below the specified minimum version, it will be required to update to the latest version during ADE before device setup and enrollment can proceed. You can find the latest version in the [Apple Software Lookup Service](https://gdmf.apple.com/v2/pmv). Apple's software updates are relatively large (up to several GBs) so ask your end users to connect to a Wi-Fi network that can handle large downloads during ADE.

### Windows

End users are encouraged to update Windows via the native Windows dialog.

|                                           | Before deadline | Past deadline |
| ----------------------------------------- | ----------------| ------------- |
| End user can defer automatic restart      | ✅              | ❌            |

If an end user was on vacation when the deadline passed, the end user is given a grace period (configured) before the host automatically restarts.

Fleet enforces OS updates for [quality and feature updates](https://github.com/fleetdm/fleet/blob/ca865af01312728997ea6526c548246ab98955fb/ee/server/service/mdm_profiles.go#L106). Microsoft provides documentation on [types of Windows updates](https://learn.microsoft.com/en-us/windows/deployment/update/get-started-updates-channels-tools#types-of-updates).

### macOS (below version 14.0)

End users are encouraged to update macOS (via [Nudge](https://github.com/macadmins/nudge)).

![Nudge window](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

|                                      | > 1 day before deadline | < 1 day before deadline | Past deadline         |
| ------------------------------------ | ----------------------- | ----------------------- | --------------------- |
| Nudge window frequency               | Once a day at 8pm GMT   | Once every 2 hours      | Immediately on login  |
| End user can defer                   | ✅                      | ✅                      | ❌                    |
| Nudge window is dismissible          | ✅                      | ✅                      | ❌                    |

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-10">
<meta name="articleTitle" value="Enforce OS updates">
<meta name="description" value="Learn how to manage OS updates on macOS, Windows, iOS, and iPadOS devices.">
