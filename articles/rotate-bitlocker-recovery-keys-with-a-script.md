# Rotate Windows BitLocker recovery keys with a script

After you share a host's BitLocker recovery key, for example, to help a locked-out end user, you should rotate it so that the shared key stops working. Fleet can [rotate FileVault recovery keys](https://fleetdm.com/guides/enforce-disk-encryption#rotate-disk-encryption-key) on macOS, but it can't rotate BitLocker recovery keys on Windows yet. If you rotate a key using `manage-bde` or `Add-BitLockerKeyProtector`, the key is not escrowed to Fleet, so the key stored in Fleet is outdated and will not unlock the disk.

Until Fleet supports Windows key rotation, you can use the [`rotate-bitlockerkey.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/rotate-bitlockerkey.ps1) script as a workaround. This guide shows you how to run it from Fleet. The script creates a new recovery key on the host, escrows it to Fleet, and then removes the old key. When it's done, **Show disk encryption key** in Fleet shows the new key. The script only rotates the OS volume.

> **Warning:** This script is a workaround, not a supported Fleet feature. It sends the key to Fleet through the same internal endpoint that fleetd uses, and that endpoint can change between Fleet releases. Test the script on one host after every Fleet upgrade. Official support is tracked in [fleetdm/fleet#59145](https://github.com/fleetdm/fleet/issues/59145).

## Prerequisites

- Fleet 4.95.0 or later, with `server.private_key` configured
- Windows hosts enrolled in Fleet's Windows MDM, with fleetd installed and scripts enabled
- Disk encryption turned on for the hosts' fleet in **Controls > OS settings > Disk encryption**
- The OS volume fully encrypted, with a disk encryption status of **Verified** in Fleet
- HTTPS access from the hosts to the Fleet server
- The [`rotate-bitlockerkey.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/rotate-bitlockerkey.ps1) script, downloaded from GitHub

The script doesn't support hosts that use Fleet host identity certificates. It stops before making any changes on those hosts.

## How the script works

The script detects the Fleet server URL and fleetd's install folder from the fleetd service, so you don't need to edit it. Before it changes anything, it asks Fleet for the host's disk encryption status and stops unless the status is **Verified**. This matters because Fleet accepts keys without storing them when disk encryption is off. Then it adds a new recovery key to the OS volume and sends it to Fleet. What it does next depends on Fleet's response:

| Fleet's response | What the script does |
|:---|:---|
| HTTP 204 | Removes the old recovery keys. Key has been rotated. |
| HTTP 4xx | Removes the new recovery key. Fleet's existing key still works. |
| No response, a redirect, or HTTP 5xx | Keeps both keys, so whichever key Fleet stored still works. |

The script logs every step, but never the full recovery key or fleetd's node key. The log shows only the first and last groups of the new key for verification.

## Upload the script

1. In Fleet, go to **Controls > Scripts**.
2. Upload `rotate-bitlockerkey.ps1`.

## Run the script on a test host

Run the script on one host first, and confirm the rotation before running it on more hosts.

1. Go to **Hosts** and select a Windows host.
2. On the **Host details** page, select **Actions > Run script**.
3. Find `rotate-bitlockerkey.ps1`, open its **Actions** dropdown, and select **Run**.

The script runs the next time the host checks in with Fleet.

## Verify the rotation

1. On the **Host details** page, open the script run in the host's activity feed and select the log to view its output.
2. Confirm the output ends with a `SUCCESS` line and `Finished with exit code 0`.
3. Copy the masked key from the line that starts with `Key Fleet should now show (masked)`. It looks like `123456-******-******-******-******-******-******-654321`.
4. Select **Actions > Show disk encryption key**.
5. Compare the first and last groups of the key with the masked key from the output. They should match.

## Rotate keys on more hosts

1. Go to **Hosts** and select the fleet.
2. Select the checkbox next to each host you want to rotate keys on.
3. Select **Run script** in the table header.
4. Find `rotate-bitlockerkey.ps1` and select **Run script**.
5. To check results, select the batch in the global activity feed.

> **Note:** The script skips hosts whose disk encryption status isn't **Verified**, without making changes. On hosts that are **Enforcing** or **Action required**, fleetd may already be rotating the key, and two rotations at once can remove each other's keys.

## Troubleshooting

**Output ends with "escrow outcome unknown"**

The script didn't get a clear answer from Fleet, so it kept the old and new keys. Both unlock the disk. The log shows the reason, for example `No response from Fleet` or an HTTP 5xx status. Fix the connection or the Fleet server error, then run the script again.

**Output shows "Fleet rejected the escrow"**

Fleet returned an HTTP 4xx status, so the script removed the new key. Fleet's existing key still works. An HTTP 401 means Fleet rejected the host's node key, and the host may need to enroll again. An HTTP 400 can mean the host isn't enrolled in Fleet's Windows MDM.

**Output shows "Could not detect the Fleet URL"**

The script couldn't read fleetd's settings from the `Fleet osquery` service. Set `$FleetUrl` at the top of the script to your Fleet server's HTTPS URL, upload it again, and run it again.

**Output shows "Only rotate a fully encrypted volume"**

BitLocker is still encrypting or decrypting the OS volume. Wait until the host's disk encryption status is **Verified**, then run the script again.

**Output shows "Fleet URL isn't an https:// URL"**

The script only sends keys over HTTPS. Set `$FleetUrl` to your Fleet server's HTTPS URL.

**Output shows "Fleet reports no disk encryption status"**

Disk encryption is off for the host's fleet, so Fleet wouldn't store a new key. The script made no changes. Turn on disk encryption for the fleet in **Controls > OS settings > Disk encryption**, wait until the host's status is **Verified**, then run the script again.

**Output shows "Only rotate when it's 'verified'"**

The host's disk encryption status in Fleet isn't **Verified** yet. The script made no changes. Wait until the status is **Verified**, then run the script again.

**Output shows "Could not read the host's disk encryption status from Fleet"**

The script couldn't reach Fleet, or Fleet rejected fleetd's device token, so it made no changes. An HTTP 401 usually means the token was just rotated. Run the script again.

**Output shows "Could not remove old protector"**

Fleet has the new key, but the old key still unlocks the disk, so the script exits with code 1. Run the script again. It escrows another new key and removes every older one, including the one it couldn't remove before.

## Further reading

- [Enforce disk encryption](https://fleetdm.com/guides/enforce-disk-encryption)
- [Scripts](https://fleetdm.com/guides/scripts)
- [Rotate Windows BitLocker recovery keys after they're viewed (fleetdm/fleet#59145)](https://github.com/fleetdm/fleet/issues/59145)

<meta name="articleTitle" value="Rotate Windows BitLocker recovery keys with a script">
<meta name="authorFullName" value="Deelan Sabido">
<meta name="authorGitHubUsername" value="d-sabido">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Rotate a Windows host's BitLocker recovery key with a PowerShell script and escrow the new key to Fleet.">
