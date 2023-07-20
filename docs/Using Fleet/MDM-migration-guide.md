# Migration

This section provides instructions for migrating your hosts away from your old MDM solution to Fleet.

## Requirements

1. A [deployed Fleet instance](../Deploying/Introduction.md)
2. [Fleet connected to Apple](./MDM-setup.md)

## Preparing to migrate manually enrolled hosts

1. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
2. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
3. In your old MDM solution, unenroll the hosts to be migrated. MacOS does not allow multiple MDMs to be installed at once.
4. Send [these guided instructions](#instructions-for-end-users) to your end users to complete the final few steps via Fleet Desktop.
    * Note that there will be a gap in MDM coverage between when the host is unenrolled from the old MDM and when the host turns on MDM in Fleet.

## Preparing to migrate automatically enrolled (DEP) hosts

> Automatic enrollment is available in Fleet Premium or Ultimate

1. Connect Fleet to Apple Business Manager (ABM). Learn how [here](./MDM-setup.md#apple-business-manager-abm).
2. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
3. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
4. Migrate your hosts to Fleet in ABM:
    1. In ABM, unassign the existing hosts' MDM server from the old MDM solution: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Unassign from the current MDM**, and select **Continue**.
    2. In ABM, assign these hosts' MDM server to Fleet: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Assign to the following MDM:**, select your Fleet server in the dropdown, and select **Continue**.
5. In your old MDM solution, unenroll the hosts to be migrated. MacOS does not allow multiple MDMs to be installed at once.
6. Send [these guided instructions](#instructions-for-end-users) to your end users to complete the final few steps via Fleet Desktop.
    * Note that there will be a gap in MDM coverage between when the host is unenrolled from the old MDM and when the host turns on MDM in Fleet.

## FileVault recovery keys

_Available in Fleet Premium_

When migrating from a previous MDM, end users need to take action to escrow FileVault keys to Fleet. The **My device** page in Fleet Desktop will present users with instructions to reset their key. 

To start, enforce FileVault (disk encryption) and escrow in Fleet. Learn how [here](./MDM-disk-encryption.md). 

After turning on disk encryption in Fleet, share [these guided instructions](#how-to-turn-on-disk-encryption) with your end users.

If your old MDM solution did not enforce disk encryption, the end user will need to restart or log out of the host.

If your old MDM solution did enforce disk encryption, the end user will need to reset their disk encryption key by following the prompt on the My device page and inputting their password. 

## Activation Lock Bypass codes

In Fleet, the [Activation Lock](https://support.apple.com/en-us/HT208987) feature is disabled by default for automatically enrolled (DEP) hosts.

If a host under the old MDM solution has Activation Lock enabled, we recommend asking the end user to follow these instructions to disable Activation Lock before migrating this host to Fleet: https://support.apple.com/en-us/HT208987.

This is because if the Activation Lock is enabled, you will need the Activation Lock bypass code to successfully wipe and reuse the Mac.

However, Activation Lock bypass codes can only be retrieved from the Mac up to 30 days after the device is enrolled. This means that when migrating from your old MDM solution, it’s likely that you’ll be unable to retrieve the Activation Lock bypass code.

## Migrating settings

To enforce the same settings on your macOS hosts in Fleet as you did using your old MDM solution, you can migrate these settings to Fleet to reduce manual work.

If your old MDM solution enforces FileVault, follow [these instructions](./MDM-disk-encryption.md) to enforce FileVault (disk encryption) using Fleet.

For all other settings: 
1. Check if your old MDM solution is able to export settings as .mobileconfig files. If it does, download these files. 
    * If it does not export settings, you will need to re-create the configuration profiles. Learn how to do that [here](./MDM-custom-macOS-settings.md#step-1-create-a-configuration-profile)
2. Create [teams](https://fleetdm.com/docs/using-fleet/teams) according to the needs of your organization
3. Follow the instructions to add configuration profiles to Fleet [here](./MDM-custom-macOS-settings.md#step-2-upload-configuration-profile-to-fleet).

## Instructions for end users

Your organization uses Fleet to check if all devices meet its security policies.

Fleet includes device management features (called “MDM”) that allow your IT team to change settings remotely on your Mac. This lets your organization keep your Mac up to date so you don’t have to.

Want to know what your organization can see? Read about [transparency](https://fleetdm.com/transparency).

### How to turn on MDM:

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select **Turn on MDM** the button in the yellow banner and follow the instructions. 
  - If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. 
  - If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

#### Automatic Enrollment (ADE)

1. If your device is enrolled in Apple Business Manager (ABM) and assigned to the Fleet server, the end user will receive a "Device Enrollment: &lt;organization&gt; can automatically configure your Mac." system notification within the macOS Notifications Center. 
   
2. After the end user clicks on the system notification, macOS will open the "Profiles" System Setting and ask the user to "Allow Device Enrollment: &lt;organization&gt; can automatically configure your Mac based on settings provided by your System Administrator."
  
3. If the end user does not Allow the setting, the system notification will continue to nag the end user until the setting has been allowed.
   
4. Once this setting has been approved, the MDM enrollment profile cannot be removed by the end user.

#### Manual Enrollment

1. If your device is not enrolled in Apple Business Manager (ABM), the end user will be given the option to manually download the MDM enrollment profile.
   
2. Once downloaded, the user will receive a system notification that the Device Enrollment profile has been needs to be installed in the System Settings &gt; Profiles section. 
   
2. After installation, the MDM enrollment profile can be removed by the end user at any time.
   
### How to turn on disk encryption

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, follow the disk encryption instructions in the yellow banner. 
  - If you don’t see the yellow banner, select the purple **Refetch** button at the top of the page. 
  - If you still don't see the yellow banner after a couple minutes or if the **My device** page presents you with an error, please contact your IT administrator.

<img width="1399" alt="My device page - turn on disk encryption" src="https://user-images.githubusercontent.com/5359586/229950451-cfcd2314-a993-48db-aecf-11aac576d297.png">

<meta name="pageOrderInSection" value="1501">
<meta name="title" value="MDM migration guide">
<meta name="description" value="Instructions for migrating hosts away from an old MDM solution to Fleet.">
<meta name="navSection" value="Device management">