# Mobile device management (MDM)

MDM features are not ready for production and are currently in development. These features are disabled by default.

MDM features allow you to manage macOS updates and macOS settings on your hosts.

To use MDM features you have to connect Fleet to Apple Push Certificates Portal. See how [here](#set-up).

## macOS updates

### End user macOS update reminders via Nudge

_Available in Fleet Premium_

End users can be reminded and encouraged to update macOS (via [Nudge](https://github.com/macadmins/nudge)).

When a minimum version and deadline is saved in Fleet, the end user sees the below window until their macOS version is at or above the minimum version.

To set the macOS updates settings in the UI, visit the **Controls** section and then select the **macOS updates** tab. To set the macOS updates settings programmatically, use the configurations listed [here](https://fleetdm.com/docs/using-fleet/configuration-files#mdm-macos-updates).

![Fleet's architecture diagram](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

As the deadline gets closer, Fleet provides stronger encouragement.

If the end user has more than 1 day until the deadline, the Nudge window is shown everyday. The end user can defer the update and close the window.

If there is less than 1 day, the window is shown every 2 hours. The end user can defer and close the window.

If the end user is past the deadline, Fleet shows the window and end user can't close the window until they upgrade.

### End user macOS update via built-in macOS notifications

Built-in macOS update reminders are available for all Fleet instances. To trigger these reminders, run the ["Schedule an OS update" MDM command](https://developer.apple.com/documentation/devicemanagement/schedule_an_os_update).

## macOS settings

In Fleet you can enforce settings on your macOS hosts remotely.

If you enforce disk encryption with Fleet, the disk encryption key (recovery key) will be stored in Fleet automatically. Learn how [here](#disk-encryption).

You can also enforce custom macOS settings. Learn how [here](#custom-settings).

### Disk encryption

_Available in Fleet Premium_

In Fleet, you can enforce disk encryption on your macOS hosts. Apple calls this [FileVault](https://support.apple.com/en-us/HT204837). If turned on, hosts’ disk encryption keys will be stored in Fleet.

To enforce disk encryption, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Disk encryption** page. Users with the maintainer and admin roles can access the settings pages.

2. Check the box next to **Turn on** and select **Save**.

`fleetctl` CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

> If you want to enforce disk encryption on all macOS hosts in a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](./configuration-files/README.md#teams).

2. Set the `mdm.macos_settings.enable_disk_encryption` configuration option to `true`.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

#### Viewing a disk encryption key

The disk encryption key allows you to reset a macOS host's password if you don't know it. This way, if you plan to prepare a host for a new employee, you can login to it and erase all its content and settings.

The key can be accessed by Fleet admin, maintainers, and observers. An event is tracked in the activity feed when a user views the key in Fleet.

How to view the disk encryption key:

1. Select a host on the **Hosts** page.

2. On the **Host details** page, select **Actions > Show disk encryption key**.

#### Reset a macOS host's password using the disk encryption key

How to reset a macOS host's password using the disk encryption key:

1. Restart the host. If you just unlocked a host that was locked remotely, the host will automatically restart.

2. On the Mac's login screen, enter the incorrect password three times. After the third failed login attempt, the Mac will display a prompt below the password field with the following message: "If you forgot your password, you can reset it using your Recovery Key." Select the right facing arrow at the end of this prompt.

3. Enter the disk encryption key. Note that Apple calls this "Recovery key." Learn how to find a host's disk encryption key [here in the docs](#viewing-a-disk-encryption-key).

4. The Mac will display a prompt to reset the password. Reset the password and save this password somewhere safe. If you plan to prepare this Mac for a new employee, you'll need this password to erase all content and settings on the Mac.

### Custom settings

In Fleet you can enforce custom settings on your macOS hosts using configuration profiles.

To enforce custom settings, first create configuration profiles with iMazing Profile editor and then add the profiles to Fleet.

#### Create a configuration profiles with iMazing Profile Creator

How to create a configuration profile with iMazing Profile Creator:

1. Download and install [iMazing Profile Creator](https://imazing.com/profile-editor).

2. Open iMazing Profile Creator and select macOS in the top bar. Fleet only supports enforcing settings on macOS hosts.

3. Find and choose the settings you'd like to enforce on your macOS hosts. Fleet recommends limiting the scope of the settings a single profile: only include settings from one tab in iMazing Profile Creator (ex. **Restrictions** tab). To enforce more settings, you can create and add additional profiles.

4. In iMazing Profile Creator, select the **General** tab. Enter a descriptive name in the **Name** field. When you add this profile to Fleet, Fleet will display this name in the Fleet UI.

5. In your top menu bar select **File** > **Save As...** and save your configuration profile. Make sure the file is saved as .mobileconfig.

#### Add configuration profiles to Fleet

In Fleet, you can add configuration profiles using the Fleet UI or fleetctl command-line tool.

The Fleet UI method is a good start if you're just getting familiar with Fleet.

The fleetctl CLI method enables managing configuration profiles in a git repository. This way you can enforce code review and benefit from git's change history.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Custom settings** page.

2. Select **Upload** and choose your configuration profile. After your configuration profile is uploaded to Fleet, Fleet will apply the profile on your macOS hosts. The profile will be applied to new macOS hosts that enroll to Fleet.

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

## Set up

To use MDM features, like enforcing settings and operating system version, you have to connect Fleet to Apple using Apple Push Notification service (APNs).

To use automatically enroll new Macs to Fleet, you have to connect Fleet to Apple Business Manager (ABM).

### Apple Push Notification service (APNs)

To connect Fleet to Apple, get these four files using the Fleet UI or the `fleetctl` command-line interface: An APNs certificate, APNs private key, Simple Certificate Enrollment Protocol (SCEP) certificate, and SCEP private key.

To do this, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Follow the instructions under **Apple Push Certificates Portal**.

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple --email <email> --org <org>`.

2. Follow the on-screen instructions.

> Take note of the Apple ID you use to sign into Apple Push Certificates Portal. You'll need to use the same Apple ID when renewing your APNs certificate. Apple requires that APNs certificates are renewed once every year. To renew, see the [APNs Renewal section](#apns-renewal) .

#### APNs Renewal

Apple requires that APNs certificates are renewed once every year. You can see the certificate's renewal date and other important APNs information using the Fleet UI or the `fleetctl` command-line interface:

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Look at the **Apple Push Certificates Portal** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

2. Look at the on-screen information.

How to renew the certificate if it's expired or about to expire:

1. Run the `fleetctl generate mdm-apple --email <email> --org <org>` command. Make sure you use the same Apple ID email address that you used when generating the original certificate.

2. Sign in to [Apple Push Certificates Portal](https://identity.apple.com) using the same Apple ID you used to get your original certificate. If you don't use the same Apple ID, you will have to turn MDM off and back on for all macOS hosts.

3. In the **Settings > Integrations > Mobile device management (MDM)** page, under Apple Push Certificates portal, find the serial number of your current certificate. In Apple Push Certificates Portal, click  **Renew** next to the certificate that has the matching serial number. If you don't renew and get a new certificate, you will have to turn MDM off and back on for all macOS hosts.

### Apple Business Manager (ABM)

_Available in Fleet Premium_

Connect Fleet to your ABM account to automatically enroll macOS hosts to Fleet when they’re first unboxed.

If a new macOS host that appears in ABM hasn't been unboxed, it will appear in Fleet with **MDM status** set to "Pending." These hosts will automatically enroll to the default team in Fleet. Learn how to update the default team [here](#default-team).

To connect Fleet to ABM, get these four files using the Fleet UI or the `fleetctl` command-line interface: An ABM certificate, private key and server token.

To do this, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Follow the instructions under **Apple Business Manager**.

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple-bm`.

2. Follow the on-screen instructions.

#### Default team

MacOS hosts purchases through Apple or authorized resellers will automatically enroll to the default team in Fleet when they're first unboxed. This means that Fleet will enforce the default team's settings on these hosts.

> After a host enrolls it can be transferred to a different team. Learn how [here](./Teams.md#transfer-hosts-to-a-team). Transferring a host automatically enforces the new team's settings and removes the old team's settings.

To change the default team, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. In the Apple Business Manager section, select the **Edit team** button next to **Default team**.

3. Choose a team and select **Save**.

`fleetctl` CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

2. Set the `mdm.apple_bm_default_team` configuration option to the desired team's name.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

#### ABM Renewal

The Apple Business Manager server token expires after a year or whenever the account that downloaded the token has their password changed. To renew the token, follow the [instructions documented in this FAQ](https://fleetdm.com/docs/using-fleet/faq#how-can-i-renew-my-apple-business-manager-server-token).

## Migration

Only one MDM solution can be used for MDM features, like enforcing settings, on each of your macOS hosts. This section provides instructions for migrating away from your old MDM solution so that you can use Fleet for MDM features.

To migrate hosts from your old MDM solution to Fleet you’ll first have to [deploy Fleet](../Deploying/Introduction.md), [add your hosts](./Adding-hosts.md), and [connect Fleet to Apple](#set-up).

### Manually enrolled hosts

If you have macOS hosts that were manually enrolled to your old MDM solution, you can migrate them to Fleet.

> Make sure your end users have an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.

How to migrate manually enrolled hosts:

1. In your old MDM solution, unenroll these hosts. MacOS does not allow multiple MDMs to be installed at once. This step is required to present end users with instructions to turn on MDM in Fleet.

2. The **My Device** page in Fleet Desktop will present end users with instructions to turn on MDM. Share [these guided instructions](#instructions-for-end-users) with your end users.

### Automatically enrolled (DEP) hosts

_Available in Fleet Premium_

If you have macOS hosts that were automatically enrolled to your old MDM solution, you can migrate them to Fleet.

> Make sure your end users have an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.

To check if you have hosts that were automatically enrolled, login to [Apple Business Manager](https://business.apple.com/) and select Devices.

How to migrate these hosts:

1. Connect Fleet to Apple Business Manager (ABM). Learn how [here](#apple-business-manager-abm).

2. In ABM, unassign these hosts' MDM server from the old MDM solution: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Unassign from the current MDM**, and select **Continue**.

3. In ABM, assign these hosts' MDM server to Fleet: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Assign to the following MDM:**, select your Fleet server in the dropdown, and select **Continue**.

4. In your old MDM solution, unenroll these hosts. MacOS does not allow multiple MDMs to be installed at once. This step is required to present end users with instructions to turn on MDM in Fleet.

5. The **My Device** page in Fleet Desktop will present end users with instructions to turn on MDM. Share [these guided instructions](#instructions-for-end-users) with your end users.

### FileVault recovery keys

_Available in Fleet Premium_

In Fleet, you can enforce FileVault (disk encryption) to be on. If turned on, hosts’ disk encryption keys will be stored in Fleet. Learn how [here](#disk-encryption).

During migration from your old MDM solution, disk encryption will be turned off for your macOS hosts until they are enrolled to Fleet and MDM is turned on for these hosts.

If your old MDM solution enforced disk encryption, your end users will need to reset their disk encryption key for Fleet to be able to store the key. The **My device** page in Fleet Desktop will present users with instructions to reset their key. Share [these guided instructions](#how-to-turn-on-disk-encryption) with your end users.

### Activation Lock Bypass codes

In Fleet, the [Activation Lock](https://support.apple.com/en-us/HT208987) feature is disabled by default for automatically enrolled (DEP) hosts.

If a Mac has Activation Lock enabled, we recommend asking the end user to follow these instructions to disable Activation Lock before migrating this host to Fleet: https://support.apple.com/en-us/HT208987.

This is because if the Activation Lock is enabled, you will need the Activation Lock bypass code to successfully wipe and reuse the Mac.

Activation Lock bypass codes can only be retrieved from the Mac up to 30 days after the device is enrolled. This means that when migrating from your old MDM solution, it’s likely that you’ll be unable to retrieve the Activation Lock bypass code.

### Migrate settings

To enforce the same settings on your macOS hosts in Fleet as you did using your old MDM solution, you have to migrate these settings to Fleet.

If your old MDM solution enforced FileVault, follow [these instructions](#how-to-turn-on-disk-encryption) to enforce FileVault (disk encryption) using Fleet.

For all other settings you enforced, you have to first export these settings as configuration profiles from your old MDM solution. Then, you have to add the configuration profiles to Fleet.

How to export settings as configuration profiles:

1. Check if your MDM solution has a feature that allows you to export settings as configuration profiles. If it does, make sure these configuration profiles are exported as .mobileconfig files. If it doesn't, follow the instructions to create configuration profiles using iMazing Profile Creator [here](#create-a-configuration-profiles-with-imazing-profile-creator). Use iMazing Profile Creator to replicate the settings you enforced.

2. Follow the instructions to add configuration profiles to Fleet [here](#add-configuration-profiles-to-fleet).


### Instructions for end users

Your organization uses Fleet to check if all devices meet its security policies.

Fleet includes device management features (called “MDM”) that allow your IT team to change settings remotely on your Mac. This lets your organization keep your Mac up to date so you don’t have to.

Want to know what your organization can see? Read about [transparency](https://fleetdm.com/transparency).

#### How to turn on MDM:

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select **Turn on MDM** the button and follow the instructions. If you don’t see the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. If you still don't see the **Turn on MDM** button after a couple minutes, please contact your IT administrator. If the **My device page** presents you with an error, please contact your IT administrator.

![My device page - turn on MDM](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/my-device-page-turn-on-mdm.png)

#### How to turn on disk encryption

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, follow the disk encryption instructions in the yellow banner. If you don’t see the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. If you still don't see the **Turn on MDM** button after a couple minutes, please contact your IT administrator. If the **My device page** presents you with an error, please contact your IT administrator.

![My device page - turn on disk encryption](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/my-device-page-turn-on-disk-encryption.png)

## Support

In Fleet, MDM features are supported for Macs running macOS 12 (Monterey) and higher.

<meta name="pageOrderInSection" value="1500">
