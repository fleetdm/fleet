# Provision Windows devices with Fleet, Okta, and a partner

> **Note:** This workflow isn't available yet. It depends on Okta's local account provisioning for Windows, which is coming soon. This feature creates the end user's local account from their Okta credentials, so they can sign in with their Okta username.

When a hardware partner prepares your Windows devices with a provisioning package, you can have each one arrive with Okta Verify and Okta's device certificate installed, ready for the end user to sign in with their Okta username. This guide covers only the Okta-specific steps. For the zero-touch workflow and its trade-offs, see [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot). For building and applying the package, see [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package).

Here's how it works:

1. Your partner applies the provisioning package, then completes Windows setup with a temporary admin account.
2. Fleet installs Okta Verify with a policy automation, and the Okta SCEP profile installs once Windows MDM turns on.
3. The lock screen prompts for an Okta username, and your partner shuts down and ships the device.

In Okta, the device appears as **Managed**.

## Prerequisites

- Fleet Premium, with [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup).
- Everything in the provisioning package guide's [prerequisites](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#prerequisites).
- Okta admin access, with the SCEP details from the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile#1-gather-your-okta-details): SCEP URL, static SCEP challenge, and CA thumbprint.

## Step 1: Add Okta Verify

1. In Fleet, head to **Software** and choose the fleet for the new devices.
2. Select **Add software > Fleet-maintained**, search for "Okta Verify", and select **Add** in the **Windows** column.

## Step 2: Install Okta Verify with a policy

The provisioning package builds fleetd with `--disable-setup-experience`, so Fleet's setup experience doesn't install software on these devices. Use a policy automation instead:

1. Head to **Policies**, choose the fleet for the new devices, and select **Add policy**. Use this query, which fails on hosts without Okta Verify:

```sql
SELECT 1 FROM programs WHERE name = 'Okta Verify' AND publisher = 'Okta, Inc.';
```

2. Select **Save**, target only **Windows**, then select **Save** again.
3. On the **Policies** page, select **Manage automations > Install software**.
4. Select your new policy, then choose **Okta Verify** in the dropdown and select **Save**.

Fleet installs Okta Verify the first time the device fails the policy. Learn more in the [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Step 3: Add Okta's local account provisioning

> **Note:** This Okta feature is coming soon.

Set up Okta's local account provisioning for Windows, so the lock screen prompts for the end user's Okta username.

## Step 4: Deploy the Okta SCEP certificate

Okta marks the device as managed when it finds a certificate from Okta's certificate authority (CA). Deploy it with a configuration profile:

1. Follow the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) to create the `OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, and `OKTA_CA_THUMBPRINT` variables and get the profile.
2. Head to **Controls > OS settings > Configuration profiles**, select **Add profile**, and upload the profile to the fleet for the new devices.

Windows MDM turns on after someone signs in to the device, so the profile installs once your partner signs in with the temporary admin account.

## Step 5: Build the provisioning package

Follow the provisioning package guide from [Build fleetd](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#build-fleetd) through [Export the package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#export-the-package). Use the enroll secret for the fleet from Step 1.

## Share with your partner

Send your partner:

- **The `.ppkg` file**, on a USB drive or through a secure file share.
- **The package password**, through a separate channel from the file.
- **The temporary admin account** to create during Windows setup: its username and password.
- **These instructions:**
  1. Follow [Apply the package and ship the device](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package#apply-the-package-and-ship-the-device) through the step that removes the USB drive.
  2. Instead of shutting down, finish Windows setup with the temporary admin account.
  3. At the desktop, stay signed in and connected to the internet while Fleet installs Okta Verify and the Okta certificate. Don't restart or shut down the device during this step.
  4. When the device restarts and the lock screen prompts for an Okta username, shut down the device and ship it.

## Verify

Before your partner ships the first device, confirm that it's set up correctly:

1. In Fleet, head to **Hosts**, select the fleet, and search for the device's serial number.
2. On **Host details > Policies**, confirm the Okta Verify policy from Step 2 is passing.
3. On **Host details > OS settings**, confirm the Okta SCEP profile is **Verified**.
4. In Okta, head to **Directory > Devices** and confirm the device appears with the status **Managed**.

## Further reading

- [Autopilot without Autopilot: zero-touch Windows deployment with Fleet](https://fleetdm.com/articles/autopilot-without-autopilot)
- [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package)
- [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile)
- [Automatically install software](https://fleetdm.com/guides/automatic-software-install-in-fleet)

<meta name="articleTitle" value="Provision Windows devices with Fleet, Okta, and a partner">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-23">
<meta name="description" value="Have a partner prepare Windows devices that ship enrolled in Fleet, with Okta Verify and Okta's device certificate installed.">
