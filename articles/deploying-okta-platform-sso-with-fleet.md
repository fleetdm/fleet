# Deploying Platform SSO with Okta and Fleet

Apple's Platform Single Sign-on (Platform SSO), [introduced at WWDC22](https://developer.apple.com/videos/play/wwdc2022/10045) alongside macOS Ventura, iOS 17, and iPadOS 17, enables users to sign in to their identity provider credentials once and automatically access apps and websites that require authentication through an IdP.

This guide details how to deploy Okta's macOS Platform SSO extension (Desktop Password Sync) to your Fleet macOS hosts.

If your Identity Provider (IdP) supports Platform Single Sign-on, deploying it in your environment offers a great and secure sign-in experience for your users.

Rather than your users having to enter credentials each time they sign in to an app protected by Okta, the Platform SSO extension will automatically perform the authentication and sync their local macOS password with their Okta password.

This speeds up the authentication process for your employees and enables them to use their Okta credentials to unlock their Mac.

**Important:** This feature requires the **Okta Device Access SKU** to enable Desktop Password Sync and Platform SSO functionality. Contact your Okta account representative if you need to purchase this license for your organization.

## Prerequisites

Before deploying Platform SSO with Okta, ensure you meet these requirements:

- **Okta Device Access SKU** is purchased and enabled for your organization
- Your Okta Identity Engine org is available
- The Okta Verify authenticator is set up in your org
- Your macOS computers are running a minimum of macOS 13 Ventura (macOS 14 Sonoma+ recommended for Platform SSO 2.0)
- **For macOS 14 Sonoma and later**: Device Access SCEP certificates are required
- Devices are enrolled using MDM software that supports deployment of payloads (Fleet)
- Users must have a password configured (this is different from passwordless users)
- The Platform Single Sign-on app is available for your org (contact your account representative if not visible)
- Optional: Touch ID must be set up if your org requires biometrics for user authentication
- Disable macOS password expiration with your MDM before deploying

## Configure Okta Platform Single Sign-on App

First, you'll need to set up the Platform Single Sign-on app in your Okta Admin Console:

1. Sign in to your Okta org as a super admin
2. In the Admin Console, go to **Applications** → **Browse App Catalog**
3. Search for **Platform Single Sign-on** and select the app
4. Click **Add integration**
5. Open Platform Single Sign-on from your Applications list
6. On the **General** tab, you can edit the app label or use the default label
7. On the **Sign on** tab, make note of the **Client ID** - you'll need this when creating the configuration profiles
8. On the **Assignments** tab, assign the app to individual users or groups who will use Desktop Password Sync
9. Click **Save**

Next, download Okta Verify for macOS from the Admin Console (**Settings** → **Downloads**). Don't download the Okta Verify package from the Apple App Store, as it lacks the necessary MDM integration features.

## Set Up Device Access SCEP Certificates (macOS 14+ Only)

**Note:** If you have devices running macOS 14 Sonoma or later, you must configure Device Access SCEP certificates before proceeding with Platform SSO deployment.

### Generate SCEP URL and Secret Key

1. In the Okta Admin Console, go to **Security** → **Device integrations**
2. Click the **Device Access** tab (not Endpoint management)
3. Click **Add platform**
4. Select **Desktop (Windows and macOS only)**, then click **Next**
5. On the Add device management platform page, select:
   - **Certificate authority:** Use Okta as certificate authority
   - **SCEP URL challenge type:** Static SCEP URL
6. Click **Generate**
7. **Important:** Copy and save both the SCEP URL and secret key - you'll need these for Fleet configuration
8. Click **Save**

### Create SCEP Certificate Profile in Fleet

Now create a SCEP certificate profile to deploy via Fleet:

On your Mac, open [iMazing Profile Editor](https://imazing.com/profile-editor). Create a new profile and add a **SCEP** payload with these settings:

#### Under the General tab:
- **Name:** Okta Device Access SCEP
- **Identifier**: Enter a unique string (e.g. "com.okta.device.access.53D4F816-6B96-400A-81A4-2C141E582D54")
- **UUID**: Make sure that this field is populated.

#### Under SCEP
- **URL:** The SCEP URL from Okta (step 7 above)
- **Challenge:** The secret key from Okta (step 7 above)
- **Subject:** `CN=managementAttestation %HardwareUUID%`
- **Key Size:** 2048
- **Key Usage:** Signing
- **Key is Extractable:** Unchecked
- **Allow All Apps Access:** Checked
- **Certificate Expiration Notification**: Set to 14 days before expiration.

***NOTE:*** Okta currently doesn't support automatic certificate renewal. This means you will need to redeploy the configuration profile prior to expiration.
Use the following policy to help find devices with certificates expiring:

```sql
-- Returns 1 if all Okta certs are valid for >14 days (PASSING)
-- Returns 0 if any Okta certs expire within 14 days (FAILING)
SELECT 1 
WHERE NOT EXISTS (
  SELECT 1 
  FROM certificates
  WHERE issuer LIKE '%/DC=com/DC=okta%'
    AND CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) <= 14
    AND CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) >= 0
);
```

Save this as `.mobileconfig`.

## Install Okta Verify via Fleet

On your Fleet server, select the team you want to deploy Platform SSO to. Navigate to **Software** → **Add software** → **Custom package** → **Choose file**.

Select the `OktaVerify-Installer.pkg` file on your computer, then click the **Add software** button.

Choose if you want to manually install the Okta Verify app on your hosts or have Fleet automatically do it. If you select **Automatic**, Fleet will create a policy to detect which hosts do not have the Okta Verify app and install it. If you select **Manual**, you'll need to trigger the install from the Software tab on individual hosts from the host's details page.

## Build the Platform SSO Configuration Profiles

Desktop Password Sync requires multiple configuration profiles to work properly. You'll need to create separate profiles for each component.

### 1. Associated Domains Profile

Create a new profile in iMazing Profile Editor and add an **Associated Domains** payload:

- **App Identifier:** `B7F62B65BN.com.okta.mobile.auth-service-extension`
- **Associated Domains:** `authsrv:yourdomain.okta.com` (replace with your actual Okta domain)

For macOS 15 Sequoia and later, add a second entry:
- **App Identifier:** `B7F62B65BN.com.okta.mobile`

Save as `okta-associated-domains.mobileconfig`.

### 2. Extensible Single Sign-On Profile

Create a new profile and add an **Extensible Single Sign-On** payload.

**For macOS 13 Ventura:**
- **Extension Identifier:** `com.okta.mobile.auth-service-extension`
- **Type:** Redirect
- **Team Identifier:** `B7F62B65BN`
- **URLs:** 
  - `https://yourdomain.okta.com/device-access/api/v1/nonce`
  - `https://yourdomain.okta.com/oauth2/v1/token`
- **Authentication Method:** Password

**For macOS 14 Sonoma and later:**
Same as above, but also add these Platform SSO settings:
- **Platform SSO Authentication Method:** Password
- **Use Shared Device Keys:** Checked

Example configuration for macOS 14:

```xml
<key>PayloadType</key>
<string>com.apple.extensiblesso</string>
<key>PlatformSSO</key>
<dict>
    <key>AuthenticationMethod</key>
    <string>Password</string>
    <key>UseSharedDeviceKeys</key>
    <true/>
</dict>
<key>ExtensionIdentifier</key>
<string>com.okta.mobile.auth-service-extension</string>
<key>TeamIdentifier</key>
<string>B7F62B65BN</string>
<key>Type</key>
<string>Redirect</string>
<key>URLs</key>
<array>
    <string>https://yourdomain.okta.com/device-access/api/v1/nonce</string>
    <string>https://yourdomain.okta.com/oauth2/v1/token</string>
</array>
```

Save as `okta-sso-extension.mobileconfig`.

### 3. Okta Verify App Configuration Profiles

You need to create managed app configuration profiles for two preference domains:

#### com.okta.mobile Configuration
Create a new profile and select the `Okta Verify` Application Domain: 
- **Preference Domain:** `com.okta.mobile`
- **Settings:**
  - **Okta Org Url:** `https://yourdomain.okta.com`
  - **Okta User Principle Name:** `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`

#### com.okta.mobile.auth-service-extension Configuration

**For macOS 13 Ventura:**
- **Preference Domain:** `com.okta.mobile.auth-service-extension`
- **Settings:**
  - **Okta Org Url:** `https://yourdomain.okta.com`
  - **Okta Client ID:** Your Client ID from the Platform Single Sign-on app
  - **Okta User Principle Name:**  `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`

**For macOS 14 Sonoma and later:**
Same as above, plus:
- **Platform SSO Protocol Version:** `2.0`

Save as `okta-app-config.mobileconfig`.

### 4. Security Preference Profile (Optional)

To prevent users from changing their local password (since it syncs with Okta), create a security preference profile:

- **Preference Domain:** `com.apple.preference.security`
- **Settings:**
  - **dontAllowPasswordResetUI:** `true`

Save as `okta-security-restrictions.mobileconfig`.

## Deploy Configuration Profiles via Fleet

Now deploy all the configuration profiles to your Fleet hosts:

1. On your Fleet server, select the team you want to deploy Platform SSO to
2. Navigate to **Controls** → **OS Settings** → **Custom settings**
3. Upload each profile in this order:
   - `okta-device-access-scep.mobileconfig` (macOS 14+ only)
   - `okta-associated-domains.mobileconfig`
   - `okta-sso-extension.mobileconfig`
   - `okta-app-config.mobileconfig`
   - `okta-security-restrictions.mobileconfig` (optional)

Uploading the profiles to a team in Fleet will automatically deliver them to all macOS hosts enrolled in that team. If you wish to have more control over which hosts receive the profiles, you can use labels to target or exclude specific hosts.

**Important:** For organizations with both macOS 13 and macOS 14+ devices, you'll need to create separate teams or use labels to deploy the appropriate profile versions to each macOS version.

## End User Experience

When the Okta Verify app and Platform SSO configuration profiles are deployed to a host, the end user will receive a notification that says **Registration Required: Please register with your identity provider**. You should direct your end users to interact with this notification by clicking the **Register** button that appears when they hover their mouse over the notification.

After clicking the register button in the notification, a **Platform Single Sign-On Registration** window will appear. After clicking **Continue**, the user will be prompted for the password they use to log into their Mac.

Next, they'll be prompted to sign into Okta. This is what associates the user's device to their Okta account and enables Desktop Password Sync.

If your organization requires biometrics for Okta FastPass, users will be prompted to set up Touch ID during this process.

Lastly, they'll be prompted to enable the Okta Verify app to be used as a Passkey. The notification will direct them to **System Settings** and enable the toggle next to the Okta Verify app.

Once registration is complete, the user's local macOS password will sync with their Okta password through Desktop Password Sync. Users can now:

- Use their Okta credentials to unlock their Mac at the login screen (macOS 14+ with Platform SSO 2.0)
- Experience seamless authentication to Okta-protected apps in web browsers
- No longer need to enter passwords or complete MFA challenges for Okta-protected resources

## Troubleshooting

### Platform SSO 2.0 Considerations
- Platform SSO 2.0 is only available on macOS 14 Sonoma and later
- Only one Okta account can be registered per device
- To change registered accounts, the device must be restored to factory settings

### Certificate Verification
To verify SCEP certificates were deployed correctly on macOS:
1. Open **Keychain Access**
2. Select **System** keychain
3. Confirm the client certificate and private key exist
4. Verify the certificate has a custom extension with OID `1.3.6.1.4.1.51150.13.1`

## Additional Resources

For more detailed information about configuring Okta Desktop Password Sync, see the [official Okta documentation](https://help.okta.com/oie/en-us/content/topics/oda/macos-pw-sync/configure-macos-password-sync.htm).

To create and customize configuration profiles, download [iMazing Profile Editor](https://imazing.com/profile-editor).

For SCEP certificate configuration details, see [Okta's Device Access SCEP documentation](https://help.okta.com/oie/en-us/content/topics/oda/oda-as-scep.htm).

---

*Get started with Fleet*
