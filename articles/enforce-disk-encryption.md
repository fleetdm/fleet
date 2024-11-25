# Enforce disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption for your macOS, Windows, Ubuntu Linux, and Fedora Linux hosts.

> Apple calls this [FileVault](https://support.apple.com/en-us/HT204837), Microsoft calls this [BitLocker](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/), and Linux typically uses [LUKS](https://en.wikipedia.org/wiki/Linux_Unified_Key_Setup) (Linux Unified Key Setup).

When disk encryption is enforced, hosts' disk encryption keys will be stored in Fleet.

For macOS hosts that automatically enroll, disk encryption is enforced during Setup Assistant. For Windows, disk encryption is enforced on the C: volume (default system/OS drive). On Linux, encryption enforcement involves user interaction to encrypt the device with LUKS.

## Enforce disk encryption

You can enforce disk encryption using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

#### Fleet UI:

1. In Fleet, head to the **Controls > OS settings > Disk encryption** page.

2. Choose which team you want to enforce disk encryption on by selecting the desired team in the teams dropdown in the upper left corner.

3. Check the box next to **Turn on** and select **Save**.

#### Fleet API: 

API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#update-disk-encryption-enforcement).

### Disk encryption status

In the Fleet UI, head to the **Controls > OS settings > Disk encryption** tab. You will see a table that shows the status of disk encryption on your hosts. 

* Verified: the host turned disk encryption on and sent their key to Fleet. Fleet verified with osquery. See instructions for viewing the disk encryption key [here](#view-disk-encryption-key).

* Verifying: the host acknowledged the MDM command to install the disk encryption profile. Fleet is verifying with osquery and retrieving the disk encryption key.

> It may take up to one hour for Fleet to collect and store the disk encryption keys from all hosts.

* Action required (pending): the end user must take action to turn disk encryption on or reset their disk encryption key.

* Enforcing (pending): the host will receive the MDM command to install the configuration profile when the host comes online.

* Removing enforcement (pending): the host will receive the MDM command to remove the disk encryption profile when the host comes online.

* Failed: hosts that failed to enforce disk encryption.

You can click each status to view the list of hosts for that status.

## Enforce disk encryption on Linux

To enforce disk encryption on Ubuntu Linux and Fedora Linux devices, Fleet supports Linux Unified Key Setup (LUKS) for encrypting volumes.

1. Share [this step-by-step guide](https://fleetdm.com/learn-more-about/encrypt-linux-device) with end users setting up a work computer running Ubuntu Linux or Fedora Linux.

> Note that full disk encryption can only enabled during operating system setup. If the operating system has already been installed, the end user will be required to re-install the OS to enable disk encryption.

2. Once the user encrypts the disk, Fleet will initiate a key escrow process through Fleet Desktop:
   * Fleet Desktop prompts the user to enter their current encryption passphrase.
   * A new encryption passphrase is generated and added as a LUKS keyslot for the encrypted volume.
   * The new passphrase is securely stored in Fleet's backend.

3. Fleet verifies that the encryption is complete, and the key has been escrowed. Once successful, the host's status will be updated to "Verified" in the disk encryption status table.

> Note: LUKS allows multiple passphrases for decrypting the volume. The original passphrase remains active along with the escrowed passphrase created by Fleet.


## View disk encryption key

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

> This action is logged in the activity log for security auditing purposes.

## Migrate macOS hosts

When migrating macOS hosts from another MDM solution, in order to complete the process of encrypting the hard drive and escrowing the key in Fleet, your end users must log out or restart their device.

Share [these guided instructions](https://fleetdm.com/guides/mdm-migration#how-to-turn-on-disk-encryption) with your end users.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-14">
<meta name="articleTitle" value="Enforce disk encryption">
<meta name="description" value="Learn how to enforce disk encryption on macOS, Windows, and Linux hosts and manage encryption keys with Fleet Premium.">
