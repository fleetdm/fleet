# Enforce disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption for your macOS and Windows hosts, and verify disk encryption for Ubuntu Linux, Kubuntu Linux and Fedora Linux hosts.

> Apple calls this [FileVault](https://support.apple.com/en-us/HT204837), Microsoft calls this [BitLocker](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/), and Linux typically uses [LUKS](https://en.wikipedia.org/wiki/Linux_Unified_Key_Setup) (Linux Unified Key Setup).

When disk encryption is enforced, hosts' disk encryption keys will be stored in Fleet.

For macOS hosts that automatically enroll, disk encryption is enforced during Setup Assistant. For Windows, currently disk encryption is enforced on the C: volume (default system/OS drive) only on hosts with a [TPM chip](https://support.microsoft.com/en-us/topic/what-s-a-trusted-platform-module-tpm-705f241d-025d-4470-80c5-4feeb24fa1ee). For Linux, encryption requires end user interaction.

## Enforce disk encryption

You can enforce disk encryption using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

#### Fleet UI:

1. In Fleet, head to the **Controls > OS settings > Disk encryption** page.

2. Choose which team you want to enforce disk encryption on by selecting the desired team in the teams dropdown in the upper left corner.

3. Check the box next to **Turn on** and select **Save**.

#### Fleet API: 

You can use the [Update disk encryption enforcement API endpoint](https://fleetdm.com/docs/rest-api/rest-api#update-disk-encryption-enforcement) to manage disk encryption settings via the API.

### Disk encryption status

In the Fleet UI, head to the **Controls > OS settings > Disk encryption** tab. You will see a table that shows the status of disk encryption on your hosts. 

* Verified: the host turned disk encryption on and sent their key to Fleet, and Fleet has verified the key with osquery. The [encryption key can be viewed within Fleet](#view-disk-encryption-key).

* Verifying: the host acknowledged the MDM command to install the disk encryption profile. Fleet is verifying with osquery and retrieving the disk encryption key.

> It may take up to two hours for Fleet to collect and store the disk encryption keys from all hosts.

* Action required (pending): the end user must take action to turn disk encryption on or reset their disk encryption key. 

* Enforcing (pending): the host will receive the MDM command to install the configuration profile when the host comes online.

* Removing enforcement (pending): the host will receive the MDM command to remove the disk encryption profile when the host comes online.

* Failed: hosts that failed to enforce disk encryption.

You can click each status to view the list of hosts for that status.

## Enforce disk encryption on Linux

Fleet supports Linux Unified Key Setup version 2 (LUKS2) for encrypting volumes to enforce disk encryption on Ubuntu Linux, Kubuntu Linux, and Fedora Linux hosts.

1. Share [this step-by-step guide](https://fleetdm.com/learn-more-about/encrypt-linux-device) with end users setting up a work computer running Ubuntu Linux, Kubuntu Linux or Fedora Linux.

> Note that full disk encryption can only enabled during operating system setup. If the operating system has already been installed, the end user will be required to re-install the OS to enable disk encryption.

2. Once the user encrypts the disk, Fleet will initiate a key escrow process through Fleet Desktop:
   * Fleet Desktop prompts the user to enter their current encryption passphrase.
   * A new encryption passphrase is generated and added as a LUKS keyslot for the encrypted volume.
   * The new passphrase is securely stored in Fleet.

3. Fleet verifies that the encryption is complete, and the key has been escrowed. Once successful, the host's status will be updated to "Verified" in the disk encryption status table.

> Note: LUKS allows multiple passphrases for decrypting the volume. The original passphrase remains active along with the escrowed passphrase created by Fleet.


## View disk encryption key

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

> The disk encryption key is deleted if a host is transferred to a team with disk encryption turned off. To re-escrow they key, transfer the host back to a team with disk encryption on.

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

When migrating macOS hosts from another MDM solution, in order to complete the process of encrypting the hard drive and escrowing the key in Fleet, your end users must log out or restart their Mac.

Share [these guided instructions](https://fleetdm.com/guides/mdm-migration#how-to-turn-on-disk-encryption) with your end users.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-14">
<meta name="articleTitle" value="Enforce disk encryption">
<meta name="description" value="Learn how to enforce disk encryption on macOS, Windows, and Linux hosts and manage encryption keys with Fleet Premium.">
