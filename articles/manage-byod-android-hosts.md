# Manage personal (BYOD) Android hosts

When an end user enrolls a personal Android device in Fleet, Android creates a [work profile](https://support.google.com/work/android/answer/6191949). The work profile keeps work apps and data separate from the end user's personal apps and data. Fleet manages only the work profile. This guide covers what Fleet can do on personal (BYOD) Android hosts, and which features require Fleet Premium.

For company-owned (fully-managed) Android hosts, see the [Android zero-touch enrollment guide](https://fleetdm.com/guides/android-zero-touch-enrollment).


## Prerequisites

- [Android MDM turned on](https://fleetdm.com/guides/android-mdm-setup) in Fleet. Android MDM is available in Fleet Free and Fleet Premium.
- Android devices that are [Play Protect certified](https://support.google.com/googleplay/answer/7165974) and running Android 14 or later.


## What Fleet manages on personal hosts

| Feature | Fleet Free | Fleet Premium |
|---|---|---|
| Enroll hosts with a work profile | ✅ | ✅ |
| Enroll hosts to a specific fleet | ❌ | ✅ |
| Require IdP authentication during enrollment | ❌ | ✅ |
| Configuration profiles | ✅ All hosts | ✅ Per fleet |
| Certificates and EAP-TLS Wi-Fi | ❌ | ✅ |
| Host vitals and work profile software inventory | ✅ | ✅ |
| OS vulnerabilities | ✅ | ✅ |
| Google Play Store apps in the work profile | ❌ | ✅ |
| Lock and clear passcode | ❌ | ✅ |
| Custom MDM commands | ✅ | ✅ |
| Unenroll | ✅ | ✅ |

Fleet can't see or control anything outside the work profile. On personal hosts, Fleet doesn't:

- See personal apps or data
- Collect the serial number, IMEI, MEID, phone number, or carrier
- Wipe the whole device, reboot it, or put it in lost mode
- Enforce OS updates


## Enroll hosts

1. In Fleet, go to **Hosts > Add hosts > Android**.
2. Select **Personal (BYOD)**.
3. Share the enrollment link or QR code with your end users.

The end user opens the link on their Android device, selects **Enroll**, and follows the steps to set up the work profile. When enrollment finishes, work apps appear in a separate **Work** tab in the app drawer.

> **Note:** If the device already has a work profile from another MDM solution, the end user must remove it first. Follow the [Android BYOD MDM migration guide](https://fleetdm.com/guides/android-byod-mdm-migration).

With Fleet Premium, select a fleet before you copy the enrollment link to enroll hosts to that fleet. You can also [require IdP authentication](https://fleetdm.com/guides/setup-experience#require-idp-authentication) before end users enroll.


## Apply OS settings

Use [configuration profiles](https://fleetdm.com/guides/custom-os-settings) to enforce settings in the work profile, like passcode requirements and restrictions. Android configuration profiles are JSON files that use fields from Google's Android Management API.

With Fleet Premium, you can apply different profiles to each fleet and scope profiles with labels.

You can also [deploy certificates](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate) and [connect hosts to Wi-Fi with EAP-TLS](https://fleetdm.com/guides/configure-eap-tls-wifi-android). Both require Fleet Premium.


## View host vitals and software

Fleet updates host vitals and software automatically, so there's no **Refetch** button for Android hosts. To trigger an update, the end user can [sync the device manually](https://fleetdm.com/guides/how-to-manually-sync-an-android-device).

On the **Host details** page, you'll see:

- **Vitals:** OS version, security patch level, storage, memory, hardware model, encryption status, and more. Personal hosts show an **Enrollment ID** instead of a serial number. The Enrollment ID stays the same when a host unenrolls and enrolls again.
- **Software:** Apps installed in the work profile. Personal apps aren't listed. Learn more in the [software inventory reference](https://fleetdm.com/guides/software-inventory-reference).
- **Vulnerabilities:** Android OS vulnerabilities, based on the security patch level. Fleet doesn't report vulnerabilities for Android apps.

> **Note:** Reports and policies aren't supported on Android hosts.


## Add apps to the work profile

`Applies only to Fleet Premium`

The Google Play Store in the work profile only shows apps that you add in Fleet. On Fleet Free, you can't add apps, so end users can't install apps in the work profile. Personal apps aren't affected.

To add apps, follow the [install app store apps guide](https://fleetdm.com/guides/install-app-store-apps#google-play-android). Apps you add are available for [self-service](https://fleetdm.com/guides/software-self-service) in the work profile's Google Play Store. You can also add [custom Android apps](https://fleetdm.com/guides/deploy-custom-android-app-apk) and web apps.


## Lock a host or clear its passcode

`Applies only to Fleet Premium`

On the **Host details** page, select **Actions > Lock** or **Actions > Clear passcode**. Learn more in the [lock and wipe hosts guide](https://fleetdm.com/guides/lock-wipe-hosts).

On personal hosts, both actions depend on how the end user set up their screen lock:

- **Lock:** If the end user has a separate work profile lock, Fleet locks only the work profile. Otherwise, Fleet locks the device.
- **Clear passcode:** Fleet clears only the work profile passcode.

> **Note:** The **Wipe** action isn't available on personal hosts. To remove work data, unenroll the host.


## Send MDM commands

You can send Android Management API commands to a single host with `fleetctl` or the Fleet API. Learn how in the [MDM commands guide](https://fleetdm.com/guides/mdm-commands).

> **Note:** Personal hosts don't support the `REBOOT`, `RELINQUISH_OWNERSHIP`, `START_LOST_MODE`, or `STOP_LOST_MODE` commands. Sending `LOCK` or `RESET_PASSWORD` requires Fleet Premium.


## Unenroll hosts

On the **Host details** page, select **Actions > Unenroll**. Fleet removes the work profile, including work apps, data, and OS settings. The end user's personal apps and data stay on the device.

End users can also remove the work profile themselves in the device's **Settings**. Fleet shows the host as unenrolled once Google reports that the work profile was removed.

To re-enroll a host, share the enrollment link again. You don't need to delete the host from Fleet first. Learn more in the [Android MDM setup guide](https://fleetdm.com/guides/android-mdm-setup#re-enrolling-android-hosts).


<meta name="articleTitle" value="Manage personal (BYOD) Android hosts">
<meta name="authorFullName" value="LeAnn Gove">
<meta name="authorGitHubUsername" value="Leanngove">
<meta name="publishedOn" value="[YYYY-MM-DD]">
<meta name="category" value="guides">
<meta name="description" value="Learn what Fleet can manage on personal (BYOD) Android hosts with a work profile, and which features require Fleet Premium.">
