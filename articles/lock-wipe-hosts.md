# Lock and wipe hosts

![Lock and wipe hosts](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

_Available in Fleet Premium_

In Fleet, you can lock and wipe macOS, Windows, Linux, iOS and iPadOS hosts remotely when a host might have been lost or stolen, or to remotely prepare a device to be re-deployed to another end user.

Restricting wipe for iPhones and iPads to only company-owned iPhones and iPads is coming soon.

## Lock a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to lock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** dropdown, then click  **Lock**.
4. A confirmation dialog will appear. Confirm that you want to lock the device. The host will now be marked with a "Lock pending" badge. Once the lock command is acknowledged by the host, the badge will update to "Locked".*

The best practice for iOS and iPadOS hosts is to lock the device, given one of these circumstances: 
- When an employee is offboarded, which will lock them out, and disable any further use.**
- When an employee is under investigation, reported for suspicious activity or suspected of compromise.**
- When a host is lost, and you want to find it's location.
  - This requires sending the [`DeviceLocation`](https://developer.apple.com/documentation/devicemanagement/device-location-command) command using a [custom command](https://fleetdm.com/guides/mdm-commands)

If the host's owner (employee) is leaving the company and keeping a company-owned iOS or iPadOS host, the best practice is to wipe it.

Currently, for Windows hosts that are [Microsoft Entra joined](https://learn.microsoft.com/en-us/entra/identity/devices/concept-directory-join), the best practice is to disable the end user's account in Entra and then lock the host in Fleet. This applies to all Windows hosts that [automatically enroll](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment). These hosts are Entra joined.

> **iOS and iPadOS hosts**: Locking is only available for supervised and company-owned devices.

> **Linux hosts**: The system may automatically reboot after approximately 10 seconds to complete the lock process.

## Wipe a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to wipe. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** dropdown, then click  **Wipe**.
4. Confirm that you want to wipe the device in the dialog. The host will now be marked with a "Wipe pending" badge. Once the wipe command is acknowledged by the host, the badge will update to "Wiped".

> **Important** When wiping and re-installing the operating system (OS) on a host, delete the host from Fleet before you re-enroll it. If you re-enroll without deleting, Fleet won't escrow a new disk encryption key.

> **Windows hosts** Fleet uses the [doWipeProtected](https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#dowipeprotected) command. According to Microsoft, this leaves the host [unable to boot](https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#:~:text=In%20some%20device%20configurations%2C%20this%20command%20may%20leave%20the%20device%20unable%20to%20boot.).

## Unlock a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to unlock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** menu, then click **Unlock**.
    - **macOS**: A dialog with the PIN will appear. Type the PIN into the device to unlock it.
    - **Windows, Linux, iOS and iPadOS**: The command to unlock the host will be queued and the host will unlock once it receives the command (no PIN needed).*
4. When you click **Unlock**, Windows, Linux, iOS and iPadOS hosts will be marked with an "Unlock pending" badge. Once the host is unlocked and checks back in with Fleet, the "Unlock pending" badge will be removed. macOS hosts do not have an "Unlock pending" badge as they cannot be remotely unlocked (the PIN has to be typed into the device).

> **Linux hosts**: The system will automatically reboot after approximately 10 seconds to complete the unlock process and ensure the user interface is properly restored. If the host loses connection to Fleet, the unlock process may run again, causing the host to reboot again.

## Lock and wipe using `fleetctl`

You can lock, unlock, and wipe hosts using Fleet's command-line tool `fleetctl`:

```shell
fleetctl mdm lock --host $HOST_IDENTIFIER
```

```shell
fleetctl mdm unlock --host $HOST_IDENTIFIER
```

```shell
fleetctl mdm wipe --host $HOST_IDENTIFIER
```

`$HOST_IDENTIFIER` can be any of the host identifiers: hostname, UUID, or serial number.

Add the `--help` flag to any command to learn more about how to use it.

For macOS hosts, the `mdm unlock` command will return the six-digit PIN, which must be typed into the device in order to finish unlocking it. 

*For Windows and Linux hosts, a script will run as part of the lock and unlock actions. Details for each script can be found in GitHub for [Windows](https://github.com/fleetdm/fleet/tree/main/ee/server/service/embedded_scripts/windows_lock.ps1) and [Linux](https://github.com/fleetdm/fleet/tree/main/ee/server/service/embedded_scripts/linux_lock.sh) hosts.

** Fleet is currently tracking a [known Apple bug](https://github.com/fleetdm/fleet/issues/34208), which results in Lost mode being cleared after reboot on iOS/iPadOS 26.

<meta name="articleTitle" value="Lock and wipe hosts">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-07-09">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
