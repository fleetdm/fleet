# macOS settings

In Fleet, you can enforce settings on your macOS hosts remotely by configuring .mobileconfig profiles to be sent to the hosts. Learn how [here](#custom-settings).

In addition, you can enforce disk encryption on enrolled macOS hosts. The disk encryption key (recovery key) will be escrowed in Fleet automatically. Learn how [here](#disk-encryption).

## Requirements
1. Fleet's MDM capabilities are properly configured. Learn how [here](https://fleetdm.com/docs/using-fleet/mdm-setup)
2. Hosts on which you wish to enforce settings must be enrolled to Fleet MDM. Learn how [here]() TODO
3. A Fleet user with the maintainer or admin role

## Disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption (Apple [FileVault](https://support.apple.com/en-us/HT204837)) on your macOS hosts. If turned on, host's disk encryption keys will be escrowed in Fleet.

To enforce disk encryption, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Disk encryption** page.

2. Check the box next to **Turn on** and select **Save**.

`fleetctl` CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

> If you want to enforce disk encryption on all macOS hosts in a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](./configuration-files/README.md#teams).

2. Set the `mdm.macos_settings.enable_disk_encryption` configuration option to `true`.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

> It may take up to one hour for Fleet to collect and store the disk encryption keys from all hosts.

### Viewing a disk encryption key

The disk encryption key allows you to reset a macOS host's password if the password been forgotten. This way, if you plan to prepare a host for a new employee, you can login to it and erase all its content and settings.

The key can be accessed by Fleet admin, maintainers, and observers. An event is tracked in the activity feed when a user views the key in Fleet.

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

### Reset a macOS host's password using the disk encryption key

How to reset a macOS host's password using the disk encryption key:

1. Restart the host. If you just unlocked a host that was locked remotely, the host will automatically restart.

2. On the Mac's login screen, enter the incorrect password three times. After the third failed login attempt, the Mac will display a prompt below the password field with the following message: "If you forgot your password, you can reset it using your Recovery Key." Select the right facing arrow at the end of this prompt.

3. Enter the disk encryption key. Note that Apple calls this "Recovery key." Learn how to find a host's disk encryption key [here in the docs](#viewing-a-disk-encryption-key).

4. The Mac will display a prompt to reset the password. Reset the password and save this password somewhere safe. If you plan to prepare this Mac for a new employee, you'll need this password to erase all content and settings on the Mac.

Once the new employee sets up the Mac, a new disk encryption key will be generated and escrowed if the Mac is enrolled to Fleet with disk encryption turned on.

## Custom settings

In Fleet you can enforce custom settings on your macOS hosts using configuration profiles.

To enforce custom settings, first create configuration profiles with iMazing Profile editor and then add the profiles to Fleet.

### Create a configuration profiles with iMazing Profile Creator

How to create a configuration profile with iMazing Profile Creator:

1. Download and install [iMazing Profile Creator](https://imazing.com/profile-editor).

2. Open iMazing Profile Creator and select macOS in the top bar. Fleet only supports enforcing settings on macOS hosts.

3. Find and choose the settings you'd like to enforce on your macOS hosts. Fleet recommends limiting the scope of the settings a single profile: only include settings from one tab in iMazing Profile Creator (ex. **Restrictions** tab). To enforce more settings, you can create and add additional profiles.

4. In iMazing Profile Creator, select the **General** tab. Enter a descriptive name in the **Name** field. When you add this profile to Fleet, Fleet will display this name in the Fleet UI.

5. In your top menu bar select **File** > **Save As...** and save your configuration profile. Make sure the file is saved as .mobileconfig.

### Add configuration profiles to Fleet

In Fleet, you can add configuration profiles using the Fleet UI or fleetctl command-line tool.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Custom settings** page.

2. Select **Upload** and choose your configuration profile. After your configuration profile is uploaded to Fleet, Fleet will apply the profile to all macOS hosts in the selected team. Thereafter, the profile will be applied to new macOS hosts that enroll to that team.

fleetctl CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

> If you want to add configuration profiles to all macOS hosts on a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](./configuration-files/README.md#teams).

2. Add an `mdm.macos_settings.custom_settings` key to your YAML document. This key will hold an array of paths to your configuration profiles. See the below example `config` YAML document:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_settings:
      custom_settings:
        - /path/to/configuration_profile_A.mobileconfig
        - /path/to/configuration_profile_B.mobileconfig
      ...
```

3. Run the `fleetctl apply -f <your-config-here>.yml` command to add the configuration profiles to Fleet. Note that this will override any configuration profiles added using the Fleet UI method.

<meta name="pageOrderInSection" value="1503">
<meta name="title" value="MDM macOS settings">
