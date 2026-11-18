# Provision Windows devices with Fleet, Okta, and a partner

Fleet supports zero-touch Windows deployment without Autopilot for any identity provider (IdP). Learn how in [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot). This guide covers only the Okta-specific steps. For building and applying the package, see [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package).

When a hardware partner prepares your Windows devices with a provisioning package, you can have each one arrive with Okta Verify and Okta's device certificate installed, ready for the end user to sign in with their Okta username.

> **Note:** With a [Fleet factory image](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot) instead of a provisioning package, Fleet creates the end user's local account with a username that matches Okta. A password that matches Okta is coming soon.

Here's how it works:

1. Your partner applies a provisioning package (`.ppkg`) that installs Fleet's agent (fleetd). They shut down and ship the device.
2. The end user boots up the device. Okta Verify and the Okta SCEP certificate are installed right after they connect to Wi-Fi.
3. At the login window (lock screen), the end user picks **Other** and enters their Okta username and password.

In Okta, the device appears as **Managed**.

## Prerequisites

- Fleet Premium, with [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup).
- Everything in the provisioning package guide's [prerequisites](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#prerequisites).
- Okta admin access, with the SCEP details from the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile#1-gather-your-okta-details): SCEP URL, static SCEP challenge, and CA thumbprint.

## Step 1: Add Okta Verify

1. In Fleet, head to **Software** and choose the fleet for the new devices.
2. Select **Add software > Fleet-maintained**, search for "Okta Verify", and select **Add** in the **Windows** column.
3. Under **Deploy**, check **Force install**, then select **Add software**.

Fleet creates a policy that installs Okta Verify on each host that doesn't have it. Learn more in the [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Step 2: Create the end user's local account

With a provisioning package, the end user creates their own local account during Windows setup.

To have Fleet create the account from the end user's Okta account, so their Windows username matches Okta, use a Fleet factory image instead of a provisioning package. Your OEM loads Fleet's image at the factory, instead of your partner applying a `.ppkg`. Learn how in [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot).

> **Note:** A password that matches Okta is coming soon. Today, the end user chooses a password for their new Windows account during setup.

## Step 3: Deploy the Okta SCEP certificate

Okta marks the device as managed when it finds a certificate from Okta's certificate authority (CA). Deploy it with a configuration profile:

1. Follow the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) to create the `OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, and `OKTA_CA_THUMBPRINT` variables and get the profile.
2. Head to **Controls > OS settings > Configuration profiles**, select **Add profile**, and upload the profile to the fleet for the new devices.

## Step 4: Build the provisioning package

Follow the provisioning package guide from [Build fleetd](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#build-fleetd) through [Export the package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#export-the-package). Use the enroll secret for the fleet from Step 1.

## Share with your partner

Send your partner:

- **The `.ppkg` file**, on a USB drive or through a secure file share.
- **The package password**, through a separate channel from the file.
- **These instructions:** [Apply the package and ship the device](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#apply-the-package-and-ship-the-device).

## Verify

After the first end user connects their device to Wi-Fi, confirm that it's set up correctly:

1. In Fleet, head to **Hosts**, select the fleet, and search for the device's serial number.
2. On **Host details > Policies**, confirm the Okta Verify policy from Step 1 is passing.
3. On **Host details > OS settings**, confirm the Okta SCEP profile is **Verified**.
4. In Okta, head to **Directory > Devices** and confirm the device appears with the status **Managed**.

<meta name="articleTitle" value="Provision Windows devices with Fleet, Okta, and a partner">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-23">
<meta name="description" value="Have a partner prepare Windows devices that ship enrolled in Fleet, with Okta Verify and Okta's device certificate installed.">
