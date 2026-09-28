# MDM migration guides

Fleet supports migrating hosts to Fleet from another MDM solution on macOS, iOS, iPadOS, Windows, and Android. The steps depend on the platform and, for Apple devices, whether the host is company-owned or BYOD.


## macOS, iOS, and iPadOS

Apple hosts enrolled through Apple Business can migrate to Fleet without a factory reset. See [macOS MDM migration](https://fleetdm.com/guides/mdm-migration) for the enrollment steps, migration workflows, and how to check migration progress on macOS. For iOS and iPadOS devices in Apple Business, see [Apple's documentation.](https://support.apple.com/guide/deployment/migrate-managed-devices-dep4acb2aa44/web)


## Windows

Fleet can automatically migrate Windows hosts from another MDM solution without end user interaction. See [Automatic Windows MDM migration.](https://fleetdm.com/guides/windows-mdm-setup#automatic-windows-mdm-migration)


## Android (BYOD)

BYOD Android devices enroll into Fleet through a Work Profile, so migrating means removing the old Work Profile before enrolling in Fleet. See [Android BYOD MDM migration.](https://fleetdm.com/guides/android-byod-mdm-migration)


## Linux

Fleet manages Linux hosts with fleetd rather than a traditional MDM enrollment, so there's no separate migration step. [Enroll your Linux hosts](https://fleetdm.com/guides/enroll-hosts) directly, alongside removing any existing management agent.


<meta name="articleTitle" value="MDM migration guides">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-28">
<meta name="description" value="Find the right guide for migrating macOS, iOS, iPadOS, Windows, Android, and Linux hosts from another MDM solution to Fleet.">
