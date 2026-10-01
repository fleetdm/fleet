# Android zero-touch enrollment

`Available in Fleet Premium`

Connect Fleet to the Android zero-touch portal so that company-owned Android devices automatically enroll on first boot.

> Zero-touch enrollment requires devices purchased from an [authorized zero-touch reseller](https://androidenterprisepartners.withgoogle.com/devices/) and claimed to your organization's [zero-touch customer account](https://enterprise.google.com/android/zero-touch/customers). You also need [Android MDM turned on](https://fleetdm.com/guides/android-mdm-setup) in Fleet.

## Step 1: Copy DPC extras

1. In Fleet, head to **Settings > Integrations > Mobile device management (MDM)**.
2. Under **Android zero-touch**, select **Setup**.
3. Copy the **DPC extras** JSON.

## Step 2: Create a configuration

1. Sign in to the [Android zero-touch portal](https://enterprise.google.com/android/zero-touch/customers).
2. Select **Configurations**, then select **+ Add Configuration**.
3. Enter a **Configuration name** (e.g., "Fleet").
4. For **EMM DPC**, select **Android Device Policy**.
5. Paste the DPC extras JSON into the **DPC extras** field.
6. Fill in your **Company name**, **Support email address**, and **Support phone number**.
7. Select **Add**.

## Step 3: Apply the configuration to devices

Devices claimed by your reseller appear under **Devices** in the zero-touch portal. Each device shows its current **Configuration** (or "None" if unassigned).

To apply your configuration to a device, select **Edit** next to the device and choose the configuration you created above.

To automatically apply the configuration to new devices your organization purchases in the future, set it as the default configuration.

## Step 4: Register test devices

Normally, only an authorized reseller can register devices in the zero-touch portal. 

In order to test zero-touch enrollment with devices you already own:
1. If you haven't already, in the [zero-touch portal](https://enterprise.google.com/android/zero-touch/customers), add **OEM Test Reseller** as a reseller in the **Resellers** tab by navigating to **Other resellers**, finding **OEM Test Reseller**, then hitting **Enroll**.
2. Submit [Google's device registration form](https://docs.google.com/forms/d/1zQGYyNcK1B5Q2FGF3b95Oqvs9dSAIW-lmQc_nCcc7Y8/viewform?edit_requested=true) with the device's IMEI or serial number and **OEM Test Reseller** as the Reseller. From there, apply your configuration as described above.

## Step 5: Verify

After a device is factory reset or unboxed and connected to a network: The device enrolls to Fleet and appears on the **Hosts** page.

All zero-touch-enrolled hosts enroll to the **Unassigned** fleet. Enrolling to a specific fleet is [coming soon](https://github.com/fleetdm/fleet/issues/51479).

> Zero-touch enrollment doesn't support [end-user authentication (EUA)](https://fleetdm.com/guides/setup-experience#end-user-authentication). There's no browser-based identity provider (IdP) sign-in during zero-touch provisioning, so IdP variables in certificates won't work on these hosts.

## Troubleshoot

**Device does not enter zero-touch setup**

Check the [zero-touch portal](https://enterprise.google.com/android/zero-touch/customers) to confirm the device appears under **Devices** with a configuration assigned. The device may not be claimed to your zero-touch customer account.

If your reseller entered the device's info incorrectly, Google won't recognize the device and it won't boot into zero-touch enrollment. Ask your reseller to correct the device record.

**Device enters setup but fails to enroll**

Confirm that Android MDM is turned on in Fleet (**Settings > Integrations > MDM**). If Android MDM was turned off and back on, the DPC extras have changed — copy the new JSON from Fleet and update the configuration in the zero-touch portal.

**Device skips zero-touch and boots normally**

If the device has no network connection during setup, zero-touch is skipped. The device will factory reset itself after it connects to the internet. Ensure the device has Wi-Fi or cellular during initial setup.

## Further reading

- [Android MDM setup](https://fleetdm.com/guides/android-mdm-setup)
- [Enroll hosts](https://fleetdm.com/guides/enroll-hosts)
- [Google's zero-touch enrollment overview](https://support.google.com/work/android/answer/7514005)

<meta name="articleTitle" value="Android zero-touch enrollment">
<meta name="authorFullName" value="Konstantin Sykulev">
<meta name="authorGitHubUsername" value="ksykulev">
<meta name="publishedOn" value="2026-09-28">
<meta name="category" value="guides">
<meta name="description" value="Set up Android zero-touch enrollment so company-owned devices automatically enroll to Fleet on first boot.">
