# MDM migration guides

Fleet supports migrating hosts to Fleet from another MDM solution on macOS, iOS, iPadOS, Windows, and Android. The steps depend on the platform and, for Apple devices, whether the host is company-owned or BYOD. Use the guide below that matches your platform.


## macOS, iOS, and iPadOS

Apple hosts enrolled through Apple Business or Apple School can migrate to Fleet without a factory reset. See [macOS MDM migration](https://fleetdm.com/guides/mdm-migration) for the enrollment steps, migration workflows, and how to check migration progress.


## Windows

Fleet can automatically migrate Windows hosts from another MDM solution without end user interaction. See [Automatic Windows MDM migration](https://fleetdm.com/guides/windows-mdm-setup#automatic-windows-mdm-migration).

If you're moving existing Intune configuration policies over to Fleet, the [Intune-to-Fleet CSP converter guide](https://fleetdm.com/guides/migrating-intune-policies-to-fleet-csp-converter) automates most of the translation from Intune JSON exports to Fleet's SyncML profiles.


## Android (BYOD)

BYOD Android devices enroll into Fleet through a Work Profile, so migrating means removing the old Work Profile before enrolling in Fleet. See [Android BYOD MDM migration](https://fleetdm.com/guides/android-byod-mdm-migration).


## Linux

Fleet manages Linux hosts with fleetd rather than a traditional MDM enrollment, so there's no separate migration step. [Enroll your Linux hosts](https://fleetdm.com/guides/enroll-hosts) directly, alongside removing any existing management agent.


## Replacing a Mac?

Migrating a host to Fleet is different from migrating a user's data to a new Mac. If you're replacing hardware rather than switching MDM solutions, see [Managed Migration Assistant: Mac-to-Mac migration with Fleet](https://fleetdm.com/guides/managed-migration-assistant-mac-to-mac-migration-with-fleet).


<meta name="articleTitle" value="MDM migration guides">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-28">
<meta name="description" value="Find the right guide for migrating macOS, Windows, Android, and Linux hosts from another MDM solution to Fleet.">
