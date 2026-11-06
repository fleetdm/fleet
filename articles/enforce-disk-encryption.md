# Enforce disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption for your macOS and Windows hosts, and escrow disk encryption keys for macOS, Windows, and Linux (Ubuntu, Kubuntu, and Fedora) hosts.

> Apple calls this [FileVault](https://support.apple.com/en-us/HT204837), Microsoft calls this [BitLocker](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/), and Linux typically uses [LUKS](https://en.wikipedia.org/wiki/Linux_Unified_Key_Setup) (Linux Unified Key Setup).

When disk encryption is enforced, hosts' disk encryption keys will be stored in Fleet.

For macOS hosts that automatically enroll, end users are forced to enable disk encryption during Setup Assistant and the disk encryption key is automatically escrowed to Fleet. For hosts that manually enroll, end users are forced to enable disk encryption. The key gets escrowed the next time they log out and log back in. For both enroll methods, end users can't defer.

> On macOS 15.7, if the end user account type is set to **Standard** or **Skip (no account)** during [setup experience](https://fleetdm.com/guides/setup-experience), FileVault cannot be enabled locally through System Settings. To encrypt the disk on these hosts, enforce disk encryption via Fleet using the steps below. This issue does not affect macOS 26.

> Fleet offers two options for macOS disk encryption:
  - Enable disk encryption: Fleet enforces FileVault and escrows the recovery key.
  - Escrow recovery key: Fleet stores the recovery key but does not prompt users to turn on FileVault. Use when a third-party tool (e.g. XCreds) handles enforcement.

For Windows, currently disk encryption is enforced on the C: volume (default system/OS drive) only on hosts with a [TPM chip](https://support.microsoft.com/en-us/topic/what-s-a-trusted-platform-module-tpm-705f241d-025d-4470-80c5-4feeb24fa1ee). For Linux, encryption requires end user interaction.

## End user interaction

- **Windows**: Automatic. Fleet turns on BitLocker and escrows the recovery key without any end user action. If you require a BitLocker PIN, end users are prompted to set one on their **My device** page.
- **macOS**: Usually automatic. Hosts that automatically enroll turn on FileVault during Setup Assistant, and the key is escrowed right away. Hosts that manually enroll, or that are [migrated](#migrate-macos-hosts) from another MDM solution, need the end user to log out and log back in before Fleet can escrow the key.
- **Linux**: Requires the end user. The disk must be encrypted during OS setup, and Fleet Desktop prompts the end user to enter their disk password so Fleet can escrow a key. [Learn more](#escrow-disk-encryption-key-on-linux).

## Enforce disk encryption

You can enforce disk encryption using the Fleet UI, Fleet API, or [GitOps](https://fleetdm.com/docs/configuration/yaml-files).

#### Fleet UI:

1. In Fleet, head to the **Controls > OS settings > Disk encryption** page.

2. Choose which fleet you want to enforce disk encryption on by selecting the desired fleet in the fleets dropdown in the upper left corner.

3. Under macOS, check **Enable disk encryption (FileVault)** and/or **Escrow recovery key**.

4. Under Windows, check **Enable disk encryption (BitLocker) and escrow recovery key** and optionally **Require BitLocker PIN**.

5. Under Linux, check **Escrow recovery key**.

6. Select **Save**.

#### Fleet API: 

You can use the [Update disk encryption enforcement API endpoint](https://fleetdm.com/docs/rest-api/rest-api#update-disk-encryption-enforcement) to manage disk encryption settings via the API.

### Disk encryption status

In the Fleet UI, head to the **Controls > OS settings > Disk encryption** tab. You will see a table that shows the status of disk encryption on your hosts. 

* Verified: the host turned disk encryption on and sent their key to Fleet, and Fleet has verified the key with osquery. The [encryption key can be viewed within Fleet](#view-disk-encryption-key).

* Verifying: the host acknowledged the MDM command to install the disk encryption profile. Fleet is verifying with osquery and retrieving the disk encryption key. On Windows, this means the encryption key has been escrowed to Fleet but disk encryption may still be physically completing in the background. The host will transition to "Verified" once osquery confirms encryption is complete and BitLocker protection is active.

> It may take up to two hours for Fleet to collect and store the disk encryption keys from all hosts.

* Action required (pending): the end user must take action to turn disk encryption on or reset their disk encryption key. On Windows, this status also appears when BitLocker protection is off and Fleet couldn't turn it back on. The host's disk encryption details give the reason. If the end user can resolve it, they'll be asked to on their **My device** page, either to restart the host or to create a BitLocker PIN.

* Enforcing (pending): the host will receive the MDM command to install the configuration profile when the host comes online. On Windows, this status also appears while Fleet is turning BitLocker protection back on for a disk that's already encrypted, and while Fleet waits for a host to escrow a new key because it can't read the stored one. [Learn more](#unreadable-recovery-keys-on-windows).

* Removing enforcement (pending): the host will receive the MDM command to remove the disk encryption profile when the host comes online.

* Failed: hosts that failed to enforce disk encryption.

You can click each status to view the list of hosts for that status.

## Escrow disk encryption key on Linux

Fleet supports Linux Unified Key Setup version 2 (LUKS2) for escrowing disk encryption keys on Ubuntu Linux, Kubuntu Linux, and Fedora Linux hosts. Fleet doesn't support key escrow on Arch Linux or Arch-based distributions like Manjaro and CachyOS.

1. Share [this step-by-step guide](https://fleetdm.com/learn-more-about/encrypt-linux-device) with end users setting up a work computer running Ubuntu Linux, Kubuntu Linux or Fedora Linux.

> Note that full disk encryption can only enabled during operating system setup. If the operating system has already been installed, the end user will be required to re-install the OS to enable disk encryption.

2. Once the user encrypts the disk, Fleet will initiate a key escrow process through Fleet Desktop:
   * Fleet Desktop prompts the user to enter their current encryption passphrase.
   * A new encryption passphrase is generated and added as a LUKS keyslot for the encrypted volume.
   * The new passphrase is securely stored in Fleet.

3. Fleet verifies that the encryption is complete, and the key has been escrowed. Once successful, the host's status will be updated to "Verified" in the disk encryption status table.

> Note: LUKS allows multiple passphrases for decrypting the volume. The original passphrase remains active along with the escrowed passphrase created by Fleet.

### Remove the end user's passphrase

If you remove the end user's passphrase, only the escrowed key unlocks the disk. Someone with the disk and the end user's old passphrase can't decrypt or mount it.

1. Run `sudo cryptsetup luksDump <device>` to list the key slots. On a standard install, the end user's passphrase is in slot 0 and Fleet's key is in slot 1.
2. Save the escrowed key from **Actions > Show disk encryption key** to a file that only root can read.
3. Run `sudo cryptsetup luksKillSlot <device> <slot> --key-file <file>`, using the end user's slot.

After you remove the end user's passphrase:

- The host asks for a passphrase every time it starts, and only the escrowed key unlocks it. The end user has to contact IT after every restart. The host can't finish restarting on its own, for example after an update or a power loss.
- Fleet doesn't detect the change. The disk encryption status stays "Verified".
- A saved copy of the LUKS header, for example from a backup tool, can bring the old passphrase back.
- Don't remove Fleet's key slot. Fleet then shows "Action required" and asks the end user to create a new key.

This applies to disks that unlock only with passphrases, which is the case on a standard Ubuntu or Fedora install. If the disk also unlocks with a TPM or a key file, removing the passphrase doesn't lock the end user out.

## Create a BitLocker PIN on Windows

When a fleet requires a BitLocker PIN, Windows hosts stay in **Action required** until the end user creates one. The PIN is required at startup to unlock the disk, and the end user doesn't need administrator rights to create it.

1. Fleet Desktop shows a notification once per Windows login: "Set your BitLocker PIN to protect this device."

2. Selecting **Create PIN** opens the host's **My device** page. The end user can also get there from the **Disk encryption** banner on that page.

3. The end user enters the PIN twice and selects **Save**. A PIN is 6 to 20 characters.

4. Fleet stores the PIN encrypted and hands it to fleet's agent (fleetd) the next time the host checks in, which is within 30 seconds. fleetd applies it and Fleet deletes its copy. Fleet never shows the PIN again, to the end user or to an admin.

5. The **My device** page reports the result. Disk encryption moves to **Verified** once the host confirms the PIN.

If the end user's device is running a version of fleetd that's too old to set a PIN, **Create PIN** instead shows instructions for creating one through Windows' **Manage BitLocker**, which does require administrator rights.

## View disk encryption key

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

> The disk encryption key is deleted if a host is assigned to a fleet with disk encryption turned off. To re-escrow they key, transfer the host back to a fleet with disk encryption on.

## Rotate disk encryption key

> Available in Fleet Premium. macOS hosts only.

You can rotate a host's FileVault recovery key on demand. Fleet sends an MDM command to the host and escrows the new key as soon as the host acknowledges it. The host doesn't need a user logged in.

Requirements:
- The host is enrolled in Fleet MDM.
- Key escrow is turned on for the host's fleet.
- The host's current key is escrowed and decryptable by Fleet.

#### Fleet UI:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

3. In the disk encryption key modal, select **Rotate key**.

The button shows **Rotating...** until the host acknowledges the command. Until then, Fleet still returns the current key. A second rotation attempt while one is pending returns an error.

#### Fleet API:

Use the [Rotate host's disk encryption key API endpoint](https://fleetdm.com/docs/rest-api/rest-api#rotate-hosts-disk-encryption-key):

`POST /api/v1/fleet/hosts/:id/encryption_key/rotate`

> If the host reports that the rotation failed, Fleet logs a "Failed to rotate disk encryption key" activity and the host's disk encryption status becomes **Action required**. At the next login, the end user is prompted for their password so Fleet can escrow a new key (see [Escrow Buddy](#escrow-buddy)). No rotation is left pending.

## Use disk encryption key to login

Disk encryption keys are used to login to workstations (hosts) when the end user forgets their password or when the host is returned to the organization after an end user leaves. 

### macOS

1. With the macOS host in front of you, restart the host and select the end user's account.

2. Select the question mark icon **(?)** next to the password field and select **Restart and show password reset options**. If you don't see the **(?)** icon, try entering any incorrect password several times.

3. Follow the instructions on the Mac to enter the disk encryption (recovery) key.

### Windows

For Windows hosts, you don't need the disk encryption key.

First, in Fleet, head to the host's **Host details** page in Fleet and check it's **MDM status**. If it has an **On (automatic)** status follow the first set of instructions below. If it has an **On (manual)** status follow the second set of instructions.

#### On (automatic)

1. Login to [Microsoft Azure](https://portal.azure.com) (Entra) and navigate to the **Users** page.

2. Select the end user's user and select **Reset password**.

3. Use the new password to login to the Windows workstation.

#### On (manual)

1. Add [this script](https://github.com/fleetdm/fleet/tree/main/it-and-security/lib/windows/scripts/create-admin-user.ps1) to Fleet (creates a local admin user).

2. Head to the Windows host's **Host details** page and select **Actions > Run script** to run the script.

3. With the Windows host in front of you, restart the host and login with the new admin user.

### Linux 

1. With the Linux host in front of you, restart it.

2. When prompted to unlock the disk, enter the disk encryption key.

3. On the **Host details** page in Fleet, find the local user's username in the **Users** table.

4. Next, add the following script to Fleet (deletes the local password (passphrase)):

```
passwd -d <username>
```

5. Head back to the **Host details** page and select **Actions > Run script** to run the script.

## Migrate macOS hosts

When migrating macOS hosts from another MDM solution, in order to complete the process of encrypting the hard drive and escrowing the key in Fleet, your end users must log out and log back in.

Share [these guided instructions](https://fleetdm.com/guides/mdm-migration#how-to-turn-on-disk-encryption) with your end users.

## Advanced

### Custom disk encryption profiles

If you need FileVault or BitLocker settings Fleet doesn't manage directly, you can upload your own configuration profile alongside or instead of Fleet's enforcement. See [Custom disk encryption profiles](https://fleetdm.com/guides/custom-disk-encryption-profiles).

### BitLocker startup keys

Fleet doesn't support BitLocker startup keys, which are USB flash drives that must be plugged in to unlock the drive at startup.

On Windows, Fleet unlocks the drive with a TPM protector, and with a startup PIN if you require one. Hosts that have no TPM, and that rely on a startup key instead, aren't supported.

If a host is already encrypted with both a TPM and a startup key, Fleet leaves that setup in place when it takes over.

### Escrow Buddy

Fleet also uses [Escrow Buddy](https://github.com/macadmins/escrow-buddy) to recover from a failed disk encryption key rotation. If a rotation fails (for example, because Fleet's stored key is stale), Escrow Buddy generates a new key the next time the end user logs out and logs back in. Fleet collects the new key and the host's status returns to **Verified**.

### Encryption key changes

Currently, on macOS and Linux, Fleet detects when the disk encryption key changes and escrows a new key. Fleet doesn't do this on Windows.

On macOS hosts, if an end user with local admin permissions changes the key using the `sudo fdesetup changerecovery -personal` command, Fleet will escrow that new key.

For Linux, Fleet will prompt the end user to escrow a new key. [Learn more](#escrow-disk-encryption-key-on-linux).

On Linux, adding, changing, or removing a passphrase doesn't change the disk's encryption key or the data on the disk. The other passphrases keep working. Only re-encrypting or reformatting the disk creates a new encryption key.

### Unreadable recovery keys on Windows

Fleet encrypts Windows recovery keys with the [Fleet server's private key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key), so replacing or renewing the Windows MDM (WSTEP) certificate doesn't affect them. Before Fleet 4.95.0, Fleet encrypted them with the WSTEP certificate, and Fleet re-encrypts those keys automatically when you upgrade.

If Fleet can't read a host's stored key, the host shows "Enforcing" and its disk encryption details explain why. The next time the host comes online, it replaces its recovery key and escrows the new one, so the key Fleet couldn't read no longer unlocks the disk.


### BitLocker protection (Windows)

A Windows disk can be fully encrypted while BitLocker protection is off. This happens when something suspends BitLocker and never resumes it, such as a firmware update, an installer, or another management tool. The data stays encrypted, but the key is available in the clear, so the disk isn't protected.

Fleet turns protection back on for these hosts. While Fleet is working, the host shows "Enforcing". If Fleet can't finish, the host moves to "Action required" and the reason appears in the host's disk encryption details.

Two of those reasons are things the end user can resolve, and they'll be asked to on their **My device** page: restarting the host, when a restart is already pending and Fleet is waiting for it, and creating a BitLocker PIN, when your fleet requires one and the host doesn't have one yet. Reasons the end user can't resolve, such as a TPM that isn't ready, are shown to the admin without asking the end user to do anything.

If a policy on the host forbids the key protector Fleet needs, Fleet clears that policy so protection can be restored.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-14">
<meta name="articleTitle" value="Enforce disk encryption">
<meta name="description" value="Learn how to enforce disk encryption on macOS and Windows hosts, escrow disk encryption keys, and manage encryption keys with Fleet Premium.">
