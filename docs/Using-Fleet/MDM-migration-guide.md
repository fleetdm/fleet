# Migration

This section provides instructions for migrating your hosts away from your old MDM solution to Fleet.

## Requirements

1. A [deployed Fleet instance](../Deploying/Introduction.md)
2. [Fleet connected to Apple](./MDM-setup.md)

## Preparing to migrate manually enrolled hosts

1. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
2. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
3. In your old MDM solution, unenroll the desired hosts. MacOS does not allow multiple MDMs to be installed at once.

Once the above steps are completed, the end user now needs to complete a few steps. The **My Device** page in Fleet Desktop will present end users with instructions to turn on MDM. 

Fleet has created [these guided instructions](#instructions-for-end-users) that can be shared with your end users.

## Preparing to migrate automatically enrolled (DEP) hosts

> Automatic enrollment is available in Fleet Premium or Ultimate

1. Connect Fleet to Apple Business Manager (ABM). Learn how [here](./MDM-setup.md#apple-business-manager-abm).
2. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
3. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
4. Migrate your hosts to Fleet in ABM:
    1. In ABM, unassign the existing hosts' MDM server from the old MDM solution: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Unassign from the current MDM**, and select **Continue**.
    2. In ABM, assign these hosts' MDM server to Fleet: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Assign to the following MDM:**, select your Fleet server in the dropdown, and select **Continue**.
5. In your old MDM solution, unenroll these hosts. MacOS does not allow multiple MDMs to be installed at once. 

Once the above steps are completed, the end user now needs to complete a few steps. The **My Device** page in Fleet Desktop will present end users with instructions to turn on MDM. 

Fleet has created [these guided instructions](#instructions-for-end-users) that can be shared with your end users.

## FileVault recovery keys

_Available in Fleet Premium_

In Fleet, you can enforce FileVault (disk encryption) to be on. If turned on, hosts’ disk encryption keys will be stored in Fleet. Learn how [here](./MDM-macOS-settings.md#disk-encryption).

If your hosts did not have disk encryption turned on under the old MDM, there is no migration action needed. When you turn on disk encryption, the host will be encrypted and the key will be escrowed to Fleet automatically.

If the host had disk encryption turned on under the old MDM, disk encryption will be turned off for your macOS hosts until they are enrolled to Fleet and MDM is turned on for these hosts. Your end users will need to take an action to reset their disk encryption key for Fleet to be able to store the key. 

The **My device** page in Fleet Desktop will present users with instructions to reset their key. Share [these guided instructions](#how-to-turn-on-disk-encryption) with your end users.

## Activation Lock Bypass codes

In Fleet, the [Activation Lock](https://support.apple.com/en-us/HT208987) feature is disabled by default for automatically enrolled (DEP) hosts.

If a host under the old MDM solution has Activation Lock enabled, we recommend asking the end user to follow these instructions to disable Activation Lock before migrating this host to Fleet: https://support.apple.com/en-us/HT208987.

This is because if the Activation Lock is enabled, you will need the Activation Lock bypass code to successfully wipe and reuse the Mac.

However, Activation Lock bypass codes can only be retrieved from the Mac up to 30 days after the device is enrolled. This means that when migrating from your old MDM solution, it’s likely that you’ll be unable to retrieve the Activation Lock bypass code.

## Migrate settings

To enforce the same settings on your macOS hosts in Fleet as you did using your old MDM solution, you can migrate these settings to Fleet to reduce manual work.

If your old MDM solution enforces FileVault, follow [these instructions](#how-to-turn-on-disk-encryption) to enforce FileVault (disk encryption) using Fleet.

For all other settings: 
1. Check if your old MDM solution is able to export settings as .mobileconfig files. If it does, download these files. 
    * If it does not export settings, you will need to re-create the configuration profiles. Learn how to do that [here](./MDM-macOS-settings.md#create-a-configuration-profiles-with-imazing-profile-creator)
2. Create [teams](https://fleetdm.com/docs/using-fleet/teams) according to the needs of your organization
3. Follow the instructions to add configuration profiles to Fleet [here](./MDM-macOS-settings.md#add-configuration-profiles-to-fleet).

## Instructions for end users

Your organization uses Fleet to check if all devices meet its security policies.

Fleet includes device management features (called “MDM”) that allow your IT team to change settings remotely on your Mac. This lets your organization keep your Mac up to date so you don’t have to.

Want to know what your organization can see? Read about [transparency](https://fleetdm.com/transparency).

### How to turn on MDM:

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select **Turn on MDM** the button and follow the instructions. If you don’t see the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. If you still don't see the **Turn on MDM** button after a couple minutes, please contact your IT administrator. If the **My device page** presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

### How to turn on disk encryption

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, follow the disk encryption instructions in the yellow banner. If you don’t see the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. If you still don't see the **Turn on MDM** button after a couple minutes, please contact your IT administrator. If the **My device page** presents you with an error, please contact your IT administrator.

<img width="1399" alt="My device page - turn on disk encryption" src="https://user-images.githubusercontent.com/5359586/229950451-cfcd2314-a993-48db-aecf-11aac576d297.png">

<meta name="pageOrderInSection" value="1501">
<meta name="title" value="MDM migration guide">
