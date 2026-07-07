# Lock and wipe hosts

![Lock and wipe hosts](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

_Available in Fleet Premium_

In Fleet, you can lock and wipe macOS, Windows, Linux, iOS, iPadOS, and Android hosts remotely when a host might have been lost or stolen, or to remotely prepare a device to be re-deployed to another end user. For macOS, Windows, iOS, and iPadOS, wipe is performed via MDM commands. For Linux, wipe is [script-based](#linux-wipe-behavior).

Restricting wipe for iPhones and iPads to only company-owned iPhones and iPads is coming soon.

## Lock a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to lock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host details** page.
3. Click the **Actions** dropdown, then click  **Lock**.
4. A confirmation dialog will appear. Confirm that you want to lock the device. The host will now be marked with a "Lock pending" badge. Once the lock command is acknowledged by the host, the badge will update to "Locked".*

Currently, for Windows hosts that are [Microsoft Entra joined](https://learn.microsoft.com/en-us/entra/identity/devices/concept-directory-join), the best practice is to disable the end user's account in Entra and then lock the host in Fleet. This applies to all Windows hosts that [automatically enroll](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment). These hosts are Entra joined.

> **iOS and iPadOS**: Lock action is only available for company-owned ([supervised](https://support.apple.com/en-gb/guide/deployment/dep1d89f0bff/web)) hosts.
As part of locking an iOS or iPadOS host, Fleet collects the device's location data. Fleet will not consider the device fully locked until the location data is collected.

> **Apple hosts**: If unlocking a host within 1 minute of locking it, the host will still show the locked badge until the next MDM check-in.

> **Linux hosts**: The system may automatically reboot after approximately 10 seconds to complete the lock process.

> **Android**: The lock action will enforce the host lock screen and require the user to enter their password or PIN. It is available on company-owned and BYOD Android hosts.
>
> On a fully-managed device, it locks the whole device, while on a BYOD device, it depends on how the end user has their device lock configured. If the user has a separate work profile lock (a distinct PIN for work apps), it locks just the work profile. Android shows a **Lock pending** badge while locking, then returns to normal once acknowledged (no **Locked** badge.

### Get location of locked iOS/iPadOS host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the locked device. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host details** page
3. Under **Vitals**, click **Show location**, then click **Open in Google Maps**. This will open a new tab with the device's location shown in Google Maps.
4. While the device is locked, you can refetch device location data by clicking **Refetch**.

You can also manually send the [`DeviceLocation`](https://developer.apple.com/documentation/devicemanagement/device-location-command) command using a [custom command](https://fleetdm.com/guides/mdm-commands). This command will only work if the device is locked and in [Lost Mode](https://support.apple.com/en-gb/guide/security/secc46f3562c/web#sec49d5c5c50).

To view the location on Google Maps, use the latitude and longitude values from the command response in the following URL: `https://google.com/maps?q={latitude},{longitude}`

Example response:
```xml
  <key>Latitude</key>
  <real>37.33385013244351</real>
  <key>Longitude</key>
  <real>-122.01079213269968</real>
```

Example URL:
`https://google.com/maps?q=37.33385013244351,-122.01079213269968`

## Wipe a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to wipe. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** dropdown, then click **Wipe**.
4. Confirm that you want to wipe the device in the dialog.
   - **macOS, Windows, iOS, iPadOS**: The host will be marked with a "Wipe pending" badge. Once the wipe command is acknowledged by the host, the badge will update to "Wiped".
   - **Linux**: No "Wipe pending" or "Wiped" badge is shown. See [Linux wipe behavior](#linux-wipe-behavior) below for details.

Wiping a host silently cancels all of its upcoming activities — no canceled activity entries are added to the host's activity history.

Wiping a host silently cancels all of its upcoming activities — no canceled activity entries are added to the host's activity history.

When wiping and re-installing the operating system (OS) on a host, delete the host from Fleet before you re-enroll it. If you re-enroll without deleting, Fleet won't escrow a new disk encryption key.

If you're gifting a company-owned macOS host or you want to prevent the host from automatically re-enrolling to Fleet for some other reason, first release the host from Apple Business (AB) and then delete the host in Fleet.

For Windows hosts, Fleet uses the [doWipeProtected](https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#dowipeprotected) command by default. According to Microsoft, this leaves the host [unable to boot](https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#:~:text=In%20some%20device%20configurations%2C%20this%20command%20may%20leave%20the%20device%20unable%20to%20boot.). However, it is possible to use the [doWipe command via the API](https://fleetdm.com/docs/api/rest-api#parameters57).

If the wipe command fails (MDM protocol returns 500 in [MDM command results](https://fleetdm.com/docs/api/rest-api#list-mdm-commands)), you can run a [fallback wipe script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/wipe-windows-device.ps1) via Fleet. This script validates and repairs WinRE (the most common cause of wipe failure), suspends BitLocker, and triggers the wipe locally via the WMI-to-CSP bridge, bypassing the MDM command queue.

For macOS hosts, Fleet uses Erase All Content and Settings (EACS) with the [default fallback behavior documented by Apple](https://developer.apple.com/documentation/devicemanagement/erasedevicecommand/command-data.dictionary#:~:text=devices%20always%20obliterate.-,Default,-%3A%20If%20EACS%20preflight).

### Linux wipe behavior

> **Best practice:** Before wiping production Linux hosts, run the wipe against a test host running the same distro and version as your production hosts, with the same disk layout, filesystem configuration (including any btrfs, LVM, or LUKS setup), and network drive usage. Different distros — and even different versions of the same distro — can behave differently. Confirm the outcome matches your expectations before wiping in production.

Linux wipe does not use an MDM command because there's no standard Linux MDM protocol. Instead, Fleet runs a [script](https://github.com/fleetdm/fleet/blob/HEAD/ee/server/service/embedded_scripts/linux_wipe.sh) that does the following:

1. All non-root users are logged out and their passwords are locked.
2. Network filesystems (NFS, CIFS/SMB, SSHFS, etc.) are detected via `/proc/mounts` and unmounted before any deletion begins, to avoid accidentally erasing data on remote storage. If `/proc/mounts` is not found, the script aborts entirely rather than risk unsafe deletion. However, **the script cannot guarantee it detects every network-backed path** — symlinks that resolve outside a detected mount point or unusual mount configurations may be missed. 

> **Note:** Before wiping, ensure all network drives are disconnected from the host and verify that no critical remote data is accessible from it.

3. btrfs snapshots are attempted to be deleted — using `snapper` if available, with a fallback to `btrfs subvolume delete`. This step runs before file deletion because read-only snapshots resist `rm -rf`. Snapshot deletion may not be complete if `snapper` is not installed and the fallback cannot access all subvolumes.
4. Deletion of non-essential user data is attempted: `/home/*`, `/tmp`, `/var/tmp`, `/var/log`, user caches, trash directories, and `/.snapshots`.
5. Deletion of system directories is attempted: `/bin`, `/sbin`, `/usr`, `/lib`, `/opt`, `/etc`, `/var`, and `/srv`.
6. The host is halted via the kernel's sysrq interface.

The script will not cross filesystem boundaries — it uses `--one-file-system` (or `find -xdev` as a fallback) to avoid recursing into mounted filesystems. This is a safety measure, but it also means data on separately mounted filesystems at non-standard paths will not be erased.

#### Limitations

- This is a **best-effort, script-based** erase, not a hardware-level secure erase. There is no guarantee all data is removed. Data may be recoverable with forensic tools, particularly on SSDs without full-disk encryption.
- **Separate partitions or mount points** not covered by the paths listed above (e.g. a dedicated `/data` partition) will not be erased. If your hosts use non-standard disk layouts, the script may leave data intact on those volumes.
- **Network-mounted paths** are detected and skipped to protect remote storage, but detection relies on `/proc/mounts` being accurate and complete. **Disconnect all network drives before wiping** and confirm there is no network-backed storage accessible on the host at wipe time. If you are unsure, check what is mounted by running `mount` on the host before initiating the wipe.
- After the script completes, the host will halt and will not reboot into a usable state. Physical access and OS reinstallation will be required to bring the host back into service.

## Unlock a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to unlock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** menu, then click **Unlock**.
    - **macOS**: A dialog with the PIN will appear. Type the PIN into the device to unlock it.
    - **Windows, Linux, iOS and iPadOS**: The command to unlock the host will be queued and the host will unlock once it receives the command (no PIN needed).*
4. When you click **Unlock**, Windows, Linux, iOS and iPadOS hosts will be marked with an "Unlock pending" badge. Once the host is unlocked and checks back in with Fleet, the "Unlock pending" badge will be removed. macOS hosts do not have an "Unlock pending" badge as they cannot be remotely unlocked (the PIN has to be typed into the device).

> **Linux hosts**: The system will automatically reboot after approximately 10 seconds to complete the unlock process and ensure the user interface is properly restored. If the host loses connection to Fleet, the unlock process may run again, causing the host to reboot again.

### How to unlock offline iOS and iPadOS hosts

If an iPhone/iPad is turned off or restarted while locked, it will disconnect from Wi-Fi and can't be unlocked remotely. Connect your iPhone/iPad to your Mac with a USB and [share the network](https://support.apple.com/en-gb/guide/mac-help/mchlp1540/mac). After connecting your iPhone/iPad to the internet, in Fleet, head to the **Host details** page and select **Actions > Unlock**.

## Clear passcode on iOS, iPadOS, or Android host

You can remotely clear the passcode on an iOS, iPadOS, or Android device to help end users who have forgotten their passcode.

> Clear passcode is only available for company-owned or manually enrolled iOS/iPadOS hosts. It is not available for hosts with a personal MDM enrollment status, or hosts that are in Lost Mode or pending wipe.
> For Android hosts, the action is available for both BYOD and company-owned hosts. On a BYOD device, it removes the work profile passcode only (the user's personal device unlock is untouched). On a company-owned host, it removes the device passcode.

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the iOS or iPadOS device you want to clear the passcode for.
2. Click the host to open the **Host details** page.
3. Click the **Actions** dropdown, then click **Clear passcode**.
4. A confirmation dialog will appear. Click **Clear passcode** to confirm.

The clear passcode activity will be logged in the host's activity feed.

You can also clear the passcode using the [REST API](https://fleetdm.com/docs/api/rest-api#clear-iosipados-host-passcode) or `fleetctl`:

```http
POST /api/v1/fleet/hosts/:id/clear_passcode
```

```shell
fleetctl mdm clear-passcode --host $HOST_IDENTIFIER
```

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

For Linux hosts, running `fleetctl mdm wipe` triggers the same script-based wipe as the UI action. Read the [Linux wipe behavior](#linux-wipe-behavior) section above before using this command on production hosts.

*For Windows and Linux hosts, a script will run as part of the lock and unlock actions. Details for each script can be found in GitHub for [Windows](https://github.com/fleetdm/fleet/tree/main/ee/server/service/embedded_scripts/windows_lock.ps1) and [Linux](https://github.com/fleetdm/fleet/tree/main/ee/server/service/embedded_scripts/linux_lock.sh) hosts.

**For Linux hosts, the wipe action also runs a script. The wipe script can be found in GitHub for [Linux](https://github.com/fleetdm/fleet/tree/main/ee/server/service/embedded_scripts/linux_wipe.sh).

** Fleet is currently tracking a [known Apple bug](https://github.com/fleetdm/fleet/issues/34208), which results in Lost mode being cleared after reboot on iOS/iPadOS 26.

<meta name="articleTitle" value="Lock and wipe hosts">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-07-09">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
