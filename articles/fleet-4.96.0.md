# Fleet 4.96.0 | TODO

Fleet 4.96.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.96.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

TODO

## Changes

### Changes that may require action

- Fleet can now ask the end user to set a passcode during Android zero-touch setup, before they reach the home screen. If you set up zero-touch before Fleet 4.96.0, your configurations in the zero-touch portal use old DPC extras. Hosts still enroll to the right fleet, but the end user isn't asked for a passcode until after setup. To ask for a passcode during setup, update each configuration:
  1. In Fleet, head to **Settings > Integrations > Mobile device management (MDM)**.
  2. Under **Android zero-touch**, select **Setup**.
  3. Pick the fleet.
  4. Copy the new **DPC extras** JSON.
  5. In the [Android zero-touch portal](https://enterprise.google.com/android/zero-touch/customers), select **Configurations** and open the configuration for that fleet.
  6. Replace the **DPC extras** with the JSON you copied, then save the configuration.
  7. Repeat for each fleet that has a passcode profile.

  The old DPC extras keep working, so there's no deadline to update them. Hosts that are already enrolled aren't affected. Learn more in the [setup experience guide](https://fleetdm.com/guides/setup-experience#require-a-passcode-on-android).

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.96.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="TODO">
<meta name="articleTitle" value="Fleet 4.96.0 | TODO">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.96.0-1600x900@2x.png">
