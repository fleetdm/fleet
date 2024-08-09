# Disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption for your macOS and Windows hosts. 

> Apple calls this [FileVault](https://support.apple.com/en-us/HT204837) and Microsoft calls this [BitLocker](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/). 

When disk encryption is enforced, hosts’ disk encryption keys will be stored in Fleet.

For macOS hosts that automatically enroll, disk encryption is enforced during Setup Assistant.

For Windows, disk encryption is enforced on the C: volume (default system/OS drive).

## Enforce disk encryption

You can enforce disk encryption using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

Fleet UI:

1. In Fleet, head to the **Controls > OS settings > Disk encryption** page.

2. Choose which team you want to enforce disk encryption on by selecting the desired team in the teams dropdown in the upper left corner.

3. Check the box next to **Turn on** and select **Save**.

Fleet API: API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#update-disk-encryption-enforcement).

### Disk encryption status

In the Fleet UI, head to the **Controls > OS settings > Disk encryption** tab. You will see a table that shows the status of disk encryption on your hosts. 

* Verified: the host turned disk encryption on and sent their key to Fleet. Fleet verified with osquery. See instructions for viewing the disk encryption key [here](#view-disk-encryption-key).

* Verifying: the host acknowledged the MDM command to install the disk encryption profile. Fleet is verifying with osquery and retrieving the disk encryption key.

> It may take up to one hour for Fleet to collect and store the disk encryption keys from all hosts.

* Action required (pending): the end user must take action to turn disk encryption on or reset their disk encryption key. 

* Enforcing (pending): the host will receive the MDM command to install the configuration profile when the host comes online.

* Removing enforcement (pending): the host will receive the MDM command to remove the disk encryption profile when the host comes online.

* Failed: hosts that are failed to enforce disk encryption. 

You can click each status to view the list of hosts for that status.

## View disk encryption key

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

## Migrate macOS hosts

When migrating macOS hosts from another MDM solution, in order to complete the process of encrypting the hard drive and escrowing the key in Fleet, your end users must log out or restart their device.

Share [these guided instructions](./MDM-migration-guide.md#how-to-turn-on-disk-encryption) with your end users.

<meta name="pageOrderInSection" value="1504">
<meta name="title" value="Disk encryption">
<meta name="description" value="Learn how to enforce disk encryption on macOS and Windows hosts and manage encryption keys with Fleet Premium.">
<meta name="navSection" value="Device management">
