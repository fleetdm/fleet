# Add hosts to your Fleet Premium license

Your Fleet Premium license covers a set number of hosts. If you enroll more hosts than your license allows, Fleet shows a banner with your current count and your licensed count. This guide shows you how to add hosts to your license and apply the new license key to your Fleet server.

## Prerequisites

- A Fleet Premium license you bought on fleetdm.com. If you bought your license through Fleet's sales team, or Fleet hosts your Fleet server, [contact us](https://fleetdm.com/contact) to add hosts instead.
- The email and password for your fleetdm.com account.
- Access to your Fleet server's configuration, so you can update the `FLEET_LICENSE_KEY` environment variable or the `license.key` setting in your config file.

## Change your number of hosts

1. Log in to your [customer dashboard](https://fleetdm.com/customers/dashboard).
2. Under **Billing and payment**, select **Click here** to change your number of devices. This opens the Stripe billing portal.
3. Update the quantity to the number of hosts you need, and confirm the change.

> **Note:** If your subscription renews in the next 30 days, the dashboard asks you to [contact us](https://fleetdm.com/contact) to change your number of devices.

After Stripe processes the change, fleetdm.com generates a new license key for the new host count.

## Apply your new license key

1. Go back to your [customer dashboard](https://fleetdm.com/customers/dashboard).
2. Under **Your license**, confirm that **No. of devices** shows your new host count.
3. Select the copy icon next to **License key**.
4. Replace your existing license key with the new one, either in the `FLEET_LICENSE_KEY` environment variable or in the `license.key` setting in your [Fleet server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key).
5. Restart your Fleet server so it loads the new key.

> **Warning:** Your old license key still works until it expires, but it keeps your old host count. If you skip this step, Fleet keeps showing the banner.

## Verify

1. Log in to Fleet and reload the page.
2. Confirm that the banner about your host count is gone.

If you still have more hosts enrolled than your new license allows, the banner stays. You can raise your host count again, or delete hosts that are no longer in use from the **Hosts** page.

## Further reading

- [Fleet server configuration: license key](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key)
- [Fleet pricing](https://fleetdm.com/pricing)

<meta name="articleTitle" value="Add hosts to your Fleet Premium license">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-10-16">
<meta name="category" value="guides">
<meta name="description" value="Raise your Fleet Premium host count in the customer dashboard, then apply your new license key to your Fleet server.">
