# Disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption for your macOS and Windows hosts. 

> Apple calls this [FileVault](https://support.apple.com/en-us/HT204837) and Microsoft calls this [BitLocker](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/). 

When disk encryption is enforced, hosts’ disk encryption keys will be stored in Fleet.

## Enforce disk encryption

You can enforce disk encryption in the Fleet UI, with Fleet API, or with the fleetctl command-line interface (CLI).

Fleet UI:

1. In Fleet, head to the **Controls > OS setting > Disk encryption** page.

2. Choose which team you want to enforce disk encryption on by selecting the desired team in the teams dropdown in the upper left corner.

3. Check the box next to **Turn on** and select **Save**.

Fleet API: API documentation is [here](../REST%20API/rest-api.md#update-disk-encryption-enforcement)

`fleetctl` CLI:

1. Choose which team you want to enforce disk encryption on.

In this example, we'll enforce disk encryption on the "Workstations (canary)" team so that disk encryption only gets enforced on hosts assigned to this team.

2. Create a `workstations-canary-config.yaml` file:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      enable_disk_encryption: true        
    ...
```

To enforce settings on hosts that aren't assigned to a team ("No team"), we'll need to create an `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    enable_disk_encryption: true
  ...
```

3. Set the `mdm.enable_disk_encryption` configuration option to `true`.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command.

> Fleet auto-configures `DeferForceAtUserLoginMaxBypassAttempts` to `1`, ensuring mandatory disk encryption during new Mac setup.

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

When migrating macOS hosts another MDM solution, in order to complete the process of encrypting the hard drive and escrowing the key in Fleet, your end users must take action. 

If the host already had disk encryption turned on, the user will need to input their password. 

If the host did not already have disk encryption turned on, the user will need to log out or restart their computer.

Share [these guided instructions](./MDM-migration-guide.md#how-to-turn-on-disk-encryption) with your end users.

<meta name="pageOrderInSection" value="1504">
<meta name="title" value="Disk encryption">
<meta name="description" value="Learn how to enforce disk encryption on macOS and Windows hosts and manage encryption keys with Fleet Premium.">
<meta name="navSection" value="Device management">
