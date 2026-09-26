# Provision Windows devices with Fleet, Okta, and a partner

> **Note:** This workflow isn't available yet. It depends on Okta's local account provisioning for Windows, which is coming soon. This feature creates the end user's local account from their Okta credentials, so they can sign in with their Okta username.

When a hardware partner prepares your Windows devices, you want each one to arrive ready for its end user: enrolled in Fleet, with Okta Verify and Okta's device certificate installed, and ready to sign in with the end user's Okta username. This guide covers what you set up in Fleet and Okta, and what you share with your partner.

Here's how it works:

1. Your partner applies a provisioning package (`.ppkg`) that installs Fleet's agent (fleetd), then completes Windows setup with a temporary admin account.
2. When the device reaches the desktop, Fleet installs Okta Verify and the Okta SCEP certificate, then the device restarts.
3. The lock screen prompts for an Okta username, and your partner shuts down and ships the device.

In Okta, the device appears as **Managed**.

## Prerequisites

- Fleet Premium, with [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup).
- A fleet for the new devices, and its enroll secret. Find the secret on the **Hosts** page: select the fleet, then select **Add hosts**.
- Okta admin access, with the SCEP details from the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile#1-gather-your-okta-details): SCEP URL, static SCEP challenge, and CA thumbprint.
- Target devices running Windows 10 or 11 Pro, Enterprise, or Education. Windows Home can't turn on MDM.

## Step 1: Add Okta Verify

1. In Fleet, head to **Software** and choose the fleet for the new devices.
2. Select **Add software > Fleet-maintained**, search for "Okta Verify", and select **Add** in the **Windows** column.

## Step 2: Add Okta's local account provisioning

> **Note:** This Okta feature is coming soon.

Set up Okta's local account provisioning for Windows, so the lock screen prompts for the end user's Okta username.

## Step 3: Install Okta Verify during setup

1. Head to **Controls > Setup experience > 3. Install software** and select the **Windows** tab.
2. Select **Add software**, check **Okta Verify** and the local account provisioning from Step 2, then select **Save**.

Fleet installs both automatically as soon as fleetd enrolls the device. Learn more in the [Windows and Linux setup experience guide](https://fleetdm.com/guides/windows-linux-setup-experience#install-software).

## Step 4: Deploy the Okta SCEP certificate

Okta marks the device as managed when it finds a certificate from Okta's certificate authority (CA). Deploy it with a configuration profile:

1. Follow the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) to create the `OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, and `OKTA_CA_THUMBPRINT` variables and get the profile.
2. Head to **Controls > OS settings > Configuration profiles**, select **Add profile**, and upload the profile to the fleet for the new devices.

Windows MDM turns on after someone signs in to the device, so the profile installs once your partner signs in with the temporary admin account.

## Step 5: Build the provisioning package

Follow the [Build fleetd](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#build-fleetd), [Create the provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#create-the-provisioning-package), and [Export the package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#export-the-package) sections of the provisioning package guide. Use the enroll secret for the fleet from Step 1.

## Share with your partner

Send your partner:

- **The `.ppkg` file**, on a USB drive or through a secure file share.
- **The package password**, through a separate channel from the file.
- **The temporary admin account** to create during Windows setup: its username and password.
- **These instructions:**
  1. Apply the package at the first Windows setup screen. Follow [Apply the package and ship the device](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#apply-the-package-and-ship-the-device) through the step that removes the USB drive.
  2. Instead of shutting down, finish Windows setup with the temporary admin account.
  3. At the desktop, wait while the **Setting up your device** page installs Okta Verify and the certificate. Don't restart or shut down the device during this step.
  4. When the device restarts and the lock screen prompts for an Okta username, shut down the device and ship it.

## Verify

Before your partner ships the first device, confirm that it's set up correctly:

1. In Fleet, head to **Hosts**, select the fleet, and search for the device's serial number.
2. On **Host details > Software**, confirm Okta Verify is installed.
3. On **Host details > OS settings**, confirm the Okta SCEP profile is **Verified**.
4. In Okta, head to **Directory > Devices** and confirm the device appears with the status **Managed**.

## Further reading

- [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package)
- [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile)
- [Windows and Linux setup experience](https://fleetdm.com/guides/windows-linux-setup-experience)

<meta name="articleTitle" value="Provision Windows devices with Fleet, Okta, and a partner">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-23">
<meta name="description" value="Have a partner prepare Windows devices that ship enrolled in Fleet, with Okta Verify and Okta's device certificate installed.">
