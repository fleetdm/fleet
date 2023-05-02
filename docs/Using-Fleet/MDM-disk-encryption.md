# Disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption on your macOS hosts. Apple calls this [FileVault](https://support.apple.com/en-us/HT204837). If turned on, hosts’ disk encryption keys will be stored in Fleet.

You can also enforce custom macOS settings. Learn how [here](./MDM-custom-macOS-settings.md).

## Enforce disk encryption

To enforce disk encryption and have Fleet collect the disk encryption key, we will do the following steps:

1. Enforce disk encryption
2. Confirm disk encryption is enforced and Fleet is storing the disk encryption key

### Step 1: enforce disk encryption

To enforce disk encryption, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Disk encryption** page. Users with the maintainer and admin roles can access the settings pages.

2. Choose which team you want to enforce disk encryption on by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Check the box next to **Turn on** and select **Save**.

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
      macos_settings:
        enable_disk_encryption: true
    ...
```

To enforce settings on hosts that aren't assigned to a team ("No team"), we'll need to create an `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_settings:
      enable_disk_encryption: true
  ...
```

Learn more about configuration options for hosts that aren't assigned to a team [here](./configuration-files/README.md#organization-settings).

3. Set the `mdm.macos_settings.enable_disk_encryption` configuration option to `true`.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command.

### Step 2: confirm disk encryption is enforced and Fleet is storing the disk encryption key

1. In the Fleet UI, head to the **Controls > macOS settings > Disk encryption** tab.

2. In the table under the **Disk encryption** header click each status to view a list hosts:

* Applied: disk encryption on and key stored in Fleet. See instructions for viewing the disk encryption key [here](#view-disk-encryption-key).

* Action required (pending): the end user must take action to turn disk encryption on or reset their disk encryption key. Share [these guided instructions](./MDM-migration-guide.md#how-to-turn-on-disk-encryption) with your end users.

* Enforcing (pending): disk encryption will be enforced and the disk encryption key will be sent to Fleet when the hosts come online.

> It may take up to one hour for Fleet to collect and store the disk encryption keys from all hosts.

* Removing enforcement (pending): disk encryption enforcement will be removed when the hosts come online.

* Failed: hosts that are failed to enforce disk encryption.

## View disk encryption key

The disk encryption key allows you to reset a macOS host's password if you don't know it. This way, if you plan to prepare a host for a new employee, you can login to it and erase all its content and settings.

The key can be accessed by Fleet admin, maintainers, and observers. An event is tracked in the activity feed when a user views the key in Fleet.

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

## Reset a macOS host's password using the disk encryption key

How to reset a macOS host's password using the disk encryption key:

1. Restart the host. If you just unlocked a host that was locked remotely, the host will automatically restart.

2. On the Mac's login screen, enter the incorrect password three times. After the third failed login attempt, the Mac will display a prompt below the password field with the following message: "If you forgot your password, you can reset it using your Recovery Key." Select the right facing arrow at the end of this prompt.

3. Enter the disk encryption key. Note that Apple calls this "Recovery key." Learn how to find a host's disk encryption key [here](#view-disk-encryption-key).

4. The Mac will display a prompt to reset the password. Reset the password and save this password somewhere safe. If you plan to prepare this Mac for a new employee, you'll need this password to erase all content and settings on the Mac.

<meta name="pageOrderInSection" value="1503">
<meta name="title" value="MDM disk encryption">
