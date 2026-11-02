# Add hosts to your Fleet Premium license

If you bought Fleet Premium on fleetdm.com (self-service), follow this guide to add hosts to your license in Stripe, through your customer dashboard. 

If you bought your license through Fleet's [sales team](https://fleetdm.com/handbook/sales#team), [contact support](https://fleetdm.com/support) to add hosts instead.

## Change your number of hosts

1. Log in to your [subscription page](https://fleetdm.com/customers/update-subscription).
2. Select **Update subscription**, change your number of devices (**Quantity**), and confirm your payment.

After Stripe processes the change, fleetdm.com generates a new license key for the new host count.

## Apply your new license key

1. Go back to your [customer dashboard](https://fleetdm.com/customers/dashboard).
2. Under **Your license**, confirm that **No. of devices** shows your new host count.
3. Select the copy icon next to **License key**.
4. Replace your existing license key with the new one, either in the `FLEET_LICENSE_KEY` environment variable or in the `license.key` setting in your [Fleet server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key).
5. Restart your Fleet server so it loads the new key.

## Verify

1. Log in to Fleet and reload the page.
2. Confirm that the banner about your host count is gone.

If your enrolled hosts are still more than 20% over your new license count, the banner stays. You can raise your host count again, or delete hosts that are no longer in use from the **Hosts** page.

<meta name="articleTitle" value="Add hosts to your Fleet Premium license">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-10-16">
<meta name="category" value="guides">
<meta name="description" value="Raise your Fleet Premium host count in the customer dashboard, then apply your new license key to your Fleet server.">
