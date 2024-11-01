# Enforce OS updates

_Available in Fleet Premium_

In Fleet, you can enforce OS updates on your macOS, Windows, iOS, and iPadOS hosts remotely using the Fleet UI, Fleet API, or Fleet's GitOps workflow.

## Turning on enforcement

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

### macOS

When a minimum version is enforced, end users see a native macOS notification (DDM) once per day. Users can choose to update ahead of the deadline or schedule it for that night. 24 hours before the deadline, the notification appears hourly and ignores Do Not Disturb. One hour before the deadline, the notification appears every 30 minutes and then every 10 minutes.   

If the host was turned off when the deadline passed, the update will be scheduled an hour after it’s turned on.

For macOS devices that use Automated Device Enrollment (ADE), if the device is below the specified minimum version, it will be required to update to the latest [available version](#available-macos-ios-and-ipados-versions) during ADE before device setup and enrollment can proceed.

### iOS and iPadOS

End users will see a notification in their Notification Center after the deadline when a minimum version is enforced. They can’t use their iPhone or iPad until the OS update is installed.

For iOS and iPadOS devices that use Automated Device Enrollment (ADE), if the device is below the specified
minimum version, it will be required to update to the latest [available version](#available-macos-ios-and-ipados-versions) during ADE before device setup and enrollment can proceed.

### Available macOS, iOS, and iPadOS versions

The Apple Software Lookup Service (available at [https://gdmf.apple.com/v2/pmv](https://gdmf.apple.com/v2/pmv)) is the official resource for obtaining a list of publicly available updates, upgrades, and Rapid Security Responses. Make sure to use versions available in GDMF; otherwise, the update will not be scheduled.

### Windows

End users are encouraged to update Windows via the native Windows dialog.

|                                           | Before deadline | Past deadline |
| ----------------------------------------- | ----------------| ------------- |
| End user can defer automatic restart      | ✅              | ❌            |

If an end user was on vacation when the deadline passed, the end user is given a grace period (configured) before the host automatically restarts.

Fleet enforces OS updates for quality and feature updates. Read more about the types of Windows OS updates in the Microsoft documentation [here](https://learn.microsoft.com/en-us/windows/deployment/update/get-started-updates-channels-tools#types-of-updates).

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
