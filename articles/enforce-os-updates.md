# Enforce OS updates

_Available in Fleet Premium_

In Fleet, you can enforce OS updates on your macOS, Windows, iOS, and iPadOS hosts remotely using the Fleet UI, Fleet API, or Fleet's GitOps workflow.

For Apple (macOS, iOS, and iPadOS) hosts, Apple requires that the OS version is one from the [list of available OS versions](https://gdmf.apple.com/v2/pmv). The update will only be enforced if you use a version in that list.

For Android hosts, you can enforce OS updates using a configuration profile with the [`systemUpdate`](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#SystemUpdate) setting. This setting is only supported on fully-managed Android hosts (not BYO). Learn how to create a configuration profile in the [custom OS settings guide](https://fleetdm.com/guides/custom-os-settings).

## Fleet-managed OS updates vs. custom profiles

Fleet provides two approaches to enforce OS updates:

1. **Fleet-managed settings** — Use the Fleet UI, API, or GitOps YAML to set a minimum version and deadline. Fleet generates and deploys the appropriate enforcement profile automatically.
2. **Custom profiles** — Upload your own [Apple DDM declaration](https://developer.apple.com/documentation/devicemanagement/softwareupdateenforcementspecific) or [Windows Update CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update) profile for full control over enforcement parameters (e.g., custom enforcement time).

These two approaches are **mutually exclusive** per platform. If Fleet-managed OS update settings are configured, you cannot upload a custom OS update profile (and vice versa). You must remove one before configuring the other.

> Custom OS update profiles are a Fleet Premium feature.

## Enforce (Fleet-managed)

You can enforce OS settings using the Fleet UI, Fleet API, or [GitOps](https://fleetdm.com/docs/configuration/yaml-files).

1. Head to the **Controls** > **OS updates** tab.

2. To enforce OS updates for enrolled macOS, iOS, or iPadOS hosts, select the platform and set a **Minimum version** and **Deadline**.

3. For Windows, select **Windows** and set a **Deadline** and **Grace period**.

4. *macOS only*: check "Update new hosts to latest" if you would like hosts to automatically update to the latest OS version during automatic (ADE) enrollment, regardless of the minimum version and deadline settings.

Use the [modify fleet endpoint](https://fleetdm.com/docs/rest-api/rest-api#modify-team) to turn on minimum OS version enforcement. The relevant payload keys in the `mdm` object are:
+ `macos_updates`
+ `ios_updates`
+ `ipados_updates`
+ `windows_updates`

## GitOps

OS version enforcement options are declared within the [controls](https://fleetdm.com/docs/configuration/yaml-files#controls) section of a Fleet GitOps YAML file, using the following keys: 
+ [macos_updates](https://fleetdm.com/docs/configuration/yaml-files#macos-updates)
+ [ios_updates](https://fleetdm.com/docs/configuration/yaml-files#ios-updates)
+ [ipados_updates](https://fleetdm.com/docs/configuration/yaml-files#ipados-updates)
+ [windows_updates](https://fleetdm.com/docs/configuration/yaml-files#windows-updates)

## Custom OS update profiles

Instead of using Fleet-managed settings, you can upload a custom profile for more granular control over OS update enforcement. This is useful when you want to customize parameters that Fleet doesn't expose, such as the enforcement time (Fleet defaults to noon local time for Apple).

### Apple (macOS, iOS, iPadOS)

Upload a custom DDM declaration of type `com.apple.configuration.softwareupdate.enforcement.specific`. For example, to enforce macOS 15.4.1 with a deadline of 7 PM local time:

```json
{
  "Type": "com.apple.configuration.softwareupdate.enforcement.specific",
  "Identifier": "com.example.my-os-update-enforcement",
  "Payload": {
    "TargetOSVersion": "15.4.1",
    "TargetLocalDateTime": "2025-07-01T19:00:00"
  }
}
```

See Apple's [SoftwareUpdateEnforcementSpecific](https://developer.apple.com/documentation/devicemanagement/softwareupdateenforcementspecific) documentation for all available payload keys.

### Windows

Upload a custom Windows XML profile targeting the [Update CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update) (`./Device/Vendor/MSFT/Policy/Config/Update`). For example, to set custom deadline and grace period values:

```xml
<Atomic>
  <Replace>
    <Item>
      <Target>
        <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineForFeatureUpdates</LocURI>
      </Target>
      <Meta>
        <Type xmlns="syncml:metinf">text/plain</Type>
        <Format xmlns="syncml:metinf">int</Format>
      </Meta>
      <Data>5</Data>
    </Item>
  </Replace>
  <Replace>
    <Item>
      <Target>
        <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineForQualityUpdates</LocURI>
      </Target>
      <Meta>
        <Type xmlns="syncml:metinf">text/plain</Type>
        <Format xmlns="syncml:metinf">int</Format>
      </Meta>
      <Data>3</Data>
    </Item>
  </Replace>
  <Replace>
    <Item>
      <Target>
        <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineGracePeriod</LocURI>
      </Target>
      <Meta>
        <Type xmlns="syncml:metinf">text/plain</Type>
        <Format xmlns="syncml:metinf">int</Format>
      </Meta>
      <Data>2</Data>
    </Item>
  </Replace>
</Atomic>
```

See Microsoft's [Update CSP documentation](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update) for all available settings.

### Mutual exclusion rules

Fleet enforces the following rules to prevent conflicts between Fleet-managed settings and custom profiles:

| Action | Blocked when | Error |
| ------ | ------------ | ----- |
| Upload custom Apple OS update declaration | Fleet OS update settings are configured for that platform | "Couldn't add profile. OS updates are already configured. Remove the OS updates settings first." |
| Upload custom Windows Update CSP profile | Fleet OS update settings are configured for Windows | "Couldn't add profile. OS updates are already configured. Remove the OS updates settings first." |
| Set Fleet OS update settings (Apple) | A custom OS update declaration already exists for that platform | "Couldn't update OS updates settings. A custom OS updates declaration profile already exists. Remove the custom profile first." |
| Set Fleet OS update settings (Windows) | A custom Windows Update CSP profile already exists | "Couldn't update OS updates settings. A custom OS updates profile already exists. Remove the custom profile first." |

To switch between approaches, remove the existing configuration first, then apply the new one.

## Apple (macOS, iOS, and iPadOS) end user experience

On macOS hosts, when a minimum version is enforced, end users see a native macOS notification (DDM) once per day. Users can choose to update ahead of the deadline or schedule it for that night. 24 hours before the deadline, the notification appears hourly and ignores Do Not Disturb. One hour before the deadline, the notification appears every 30 minutes and then every 10 minutes.

> Certain user preferences may suppress macOS update notifications. To prevent users from being surprised by a forced update or unexpected restart, consider communicating OS update deadlines through additional channels.

On iOS and iPadOS hosts, end users will see a notification in their Notification Center after the deadline. They can’t use their iPhone or iPad until the OS update is installed.

If the host was turned off when the deadline passed, the update will be scheduled an hour after it’s turned on.

If you set a past date (ex. yesterday) as the deadline, the end user will immediately be prompted to install the update. If they don't, the update will automatically install in one hour. Similarly, if you set the deadline to today, end users will experience the same behavior if it's after 12 PM (end user local time).

### Update new hosts to latest

You can require hosts that automatically enroll via ADE to update to the latest version before they enroll to Fleet (during Setup Assistant).

For macOS hosts, in Fleet, head to **Controls > OS updates** and check the **Update new hosts to latest** checkbox. 

If **Update new hosts to latest** is checked, hosts below the minimum version are updated to the latest version during Setup Assistant. If a minimum version isn’t set, all hosts get updated.

For iOS/iPadOS hosts, set a minimum version and deadline. New iOS/iPadOS hosts will always update to the latest version (not the minimum version specified). On already enrolled hosts, updates are only enforced if the host is [below the minimum version](#apple-macos-ios-and-ipados-end-user-experience).

<!--

### macOS (below version 14.0)

End users are encouraged to update macOS (via [Nudge](https://github.com/macadmins/nudge)).

![Nudge window](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

|                                      | > 1 day before deadline | < 1 day before deadline | Past deadline         |
| ------------------------------------ | ----------------------- | ----------------------- | --------------------- |
| Nudge window frequency               | Once a day at 8pm GMT   | Once every 2 hours      | Immediately on login  |
| End user can defer                   | ✅                      | ✅                      | ❌                    |
| Nudge window is dismissible          | ✅                      | ✅                      | ❌                    |

-->

## Windows

End users are encouraged to update Windows via the native Windows dialog.

|                                           | Before deadline | Past deadline |
| ----------------------------------------- | ----------------| ------------- |
| End user can defer automatic restart      | ✅              | ❌            |

If an end user was on vacation when the deadline passed, the end user is given a grace period (configured) before the host automatically restarts.

Fleet enforces OS updates for [quality and feature updates](https://github.com/fleetdm/fleet/blob/ca865af01312728997ea6526c548246ab98955fb/ee/server/service/mdm_profiles.go#L106). Microsoft provides documentation on [types of Windows updates](https://learn.microsoft.com/en-us/windows/deployment/update/get-started-updates-channels-tools#types-of-updates).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-10">
<meta name="articleTitle" value="Enforce OS updates">
<meta name="description" value="Learn how to manage OS updates on macOS, Windows, iOS, and iPadOS hosts">
