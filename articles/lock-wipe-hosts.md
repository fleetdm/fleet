# Lock and wipe hosts

![Lock and wipe hosts](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

_Available in Fleet Premium_

In Fleet, you can lock and wipe macOS, Windows, and Linux hosts remotely. This allows you to easily deal with situations
where a host might have been lost or stolen, or to remotely prepare a device to be re-deployed to another end user.

**Note**: lock/unlock and wipe commands are queued and will run when the device next comes online.

## Lock a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to lock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** dropdown, then click  **Lock**.
4. A confirmation dialog will appear. Confirm that you want to lock the device. The host will now be marked with a "Lock pending" badge. Once the lock command is acknowledged by the host, the badge will update to "Locked".

## Wipe a host

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to wipe. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** dropdown, then click  **Wipe**.
4. Confirm that you want to wipe the device in the dialog. The host will now be marked with a "Wipe pending" badge. Once the wipe command is acknowledged by the host, the badge will update to "Wiped".

## Unlocking a host

**Note**: When a macOS host is locked, Fleet generates a 6 digit security PIN. This PIN must be physically input into the host in order to unlock it.

To unlock a locked host:

1. Navigate to the **Hosts** page by clicking the "Hosts" tab in the main navigation header. Find the device you want to unlock. You can search by name, hostname, UUID, serial number, or private IP address in the search box in the upper right corner.
2. Click the host to open the **Host Overview** page.
3. Click the **Actions** menu, then click **Unlock**.
    - **macOS**: A dialog with the PIN will appear. Type the PIN into the device to unlock it.
    - **Windows and Linux**: The command to unlock the host will be queued and the host will unlock once it receives the command (no PIN needed).
5. When you click **Unlock**, Windows and Linux hosts will be marked with an "Unlock pending" badge. Once the host is unlocked and checks back in with Fleet, the "Unlock pending" badge will be removed. macOS hosts do not have an "Unlock pending" badge as they cannot be remotely unlocked (the PIN has to be typed into the device).


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

**Note**: for macOS hosts, the `mdm unlock` command will return the security PIN, which must be typed into the device in order to finish unlocking it.

<meta name="articleTitle" value="Lock and wipe hosts">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-07-09">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
