# Configure Okta management attestation for mobile devices with Fleet

When an app sign-in policy requires a managed device, Okta checks whether the targeted iOS or Android device holds a key that matches one you generate in the Okta Admin Console and set in Fleet's managed app configuration. This guide shows you how to configure Okta management attestation with Fleet for iOS and Android so Okta marks your enrolled devices as managed. You can then use that status to build authentication policies that restrict app sign-ins to devices you manage.

On iOS, attestation pairs with Okta FastPass and the Credential SSO extension, so users on managed devices get passwordless, phishing-resistant single sign-on.

## Prerequisites

Check these before you start:

- Fleet with MDM turned on and configured for the platforms you'll manage (iOS, iPadOS, and Android).
- Admin access to the Okta Admin Console.
- For iOS: Volume Purchasing (VPP) connected in Fleet so you can add Okta Verify. See [install App Store apps](https://fleetdm.com/guides/install-app-store-apps).
- Your Fleet enrollment URL and enroll secret.

> **Note:** The secret key is generated per platform. Create a separate app hint in Okta for iOS and for Android.

## Configure iOS devices

### Create the platform in Okta

1. In the Admin Console, go to **Security > Device integrations**.
2. Click the **Endpoint management** tab.
3. Click **Add platform**.
4. Select **iOS**, then click **Next**.
5. Generate a new secret key or enter a custom one that meets the minimum requirements. Save the key securely to use later in Fleet.
6. For **Device management provider**, enter Fleet Device Management.
7. For **Enrollment link**, enter `https://<your-fleet-url>/enroll?enroll_secret=<your-enrollment-token>&byod=true`, replacing the values with your own.
8. Select **Yes, trust this app hint for Management Attestation now**, then click **Save**.

> **Warning:** The secret key value is shown only once. If you generate a new secret key later, update your Fleet app configuration with the new key or attestation will fail.

### Add the managed app configuration in Fleet

Okta Verify uses managed app configuration to pass the secret key that marks the device as managed.

1. Select the fleet to configure.
2. Add Okta Verify as a VPP app. See [install App Store apps](https://fleetdm.com/guides/install-app-store-apps).
3. Open the app details. Under **Library**, select the latest version of the app, then click the pencil icon to edit the app configuration.
4. Under **Configuration**, paste the following block:

```xml
<dict>
  <key>OktaVerify.UDID</key>
  <string>$FLEET_VAR_HOST_UUID</string>
  <key>managementHint</key>
  <string>your-okta-secret-key</string>
  <key>OktaVerify.OrgUrl</key>
  <string>your-okta-url</string>
</dict>
```

> **Note:** Fleet recommends storing the secret key as a `$FLEET_SECRET_*` variable instead of pasting it in plain text. See [secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

Other keys for configuring Okta Verify are available depending on your enrollment type. See Okta's [managed app configurations for iOS](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-app-configs-ios.htm).

### Configure iOS with GitOps

Create the configuration file, then reference its path from your app's YAML:

```yaml
- app_store_id: "490179405"
  platform: ios
  setup_experience: true
  auto_update_enabled: true
  configuration:
    path: ../lib/software/okta-verify-config.xml
```

## Configure Android devices

The steps for Android are mostly the same as for iOS. Create a new app hint for Android in the Okta Admin Console, following the platform steps above and selecting **Android** in place of iOS.

In Fleet, use the following configuration payload for Okta Verify:

```json
{
  "managedConfiguration": {
    "domainName": "your-okta-url",
    "managementHint": "your-okta-secret-key",
    "OktaVerify.SerialNumber": "$FLEET_VAR_HOST_HARDWARE_SERIAL"
  }
}
```

Other keys are available for the Android payload. See Okta's [managed app configurations for Android](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-app-configs-android.htm) and Fleet's [Fleet variables](https://fleetdm.com/guides/fleet-variables).

## Considerations when deploying to iOS

- On iOS, Account-driven User Enrollment supports only one instance of an app bundle on a device at a time. Apple does not allow MDM to take over management of an already-installed app. Users will need to delete and reinstall to receive the management payload.
- The whole device will be marked as managed in Okta, not only the work partition. This will allow users to access applications outside of the work profile. Consult with your SaaS vendors regarding specific controls per app. This limitation does not exist on Android.

## Further reading

- [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles)
- [Fleet variables](https://fleetdm.com/guides/fleet-variables)
- [Install App Store apps](https://fleetdm.com/guides/install-app-store-apps)

<meta name="articleTitle" value="Configure Okta management attestation for mobile devices with Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="publishedOn" value="2026-10-19">
<meta name="category" value="guides">
<meta name="description" value="Configure Okta management attestation with Fleet so Okta marks your enrolled iOS and Android devices as managed.">
