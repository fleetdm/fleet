# Provision Windows devices with Fleet, Okta, and a partner

With zero touch, new Windows devices enroll in Fleet the first time they're online, without Autopilot or Entra ID. Add a few Okta steps, and each device also shows up in Okta as **Managed**: Fleet installs Okta Verify and Okta's device certificate as soon as the device enrolls.

This guide covers only the Okta steps. To set up zero touch, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot). To learn why, see [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot).

Here's how it works:

1. Your partner or OEM prepares the device so it enrolls in Fleet the first time it's online, then ships it to the end user.
2. The end user turns on the device and connects to Wi-Fi. Fleet installs Okta Verify and the Okta SCEP certificate.
3. Okta finds the certificate and marks the device as **Managed**.

## Prerequisites

- Fleet Premium, with [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup).
- Zero touch set up for the fleet that new devices join. Learn how in [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot).
- Okta admin access, with the SCEP details from the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile#1-gather-your-okta-details): SCEP URL, static SCEP challenge, and CA thumbprint.

## Step 1: Add Okta Verify

1. In Fleet, head to **Software** and choose the fleet for the new devices.
2. Select **Add software > Fleet-maintained**, search for "Okta Verify", and select **Add** in the **Windows** column.
3. Under **Deploy**, check **Force install**, then select **Add software**.

Fleet creates a policy that installs Okta Verify on each host that doesn't have it. Learn more in the [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Step 2: Deploy the Okta SCEP certificate

Okta marks the device as managed when it finds a certificate from Okta's certificate authority (CA). Deploy it with a configuration profile:

1. Follow the [Okta Verify on Windows guide](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) to create the `OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, and `OKTA_CA_THUMBPRINT` variables and get the profile.
2. Head to **Controls > OS settings > Configuration profiles**, select **Add profile**, and upload the profile to the fleet for the new devices.

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
<meta name="description" value="Have zero-touch Windows devices show up in Okta as Managed, with Okta Verify and Okta's device certificate installed by Fleet.">
