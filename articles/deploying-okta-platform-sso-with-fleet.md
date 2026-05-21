# Deploying Platform SSO with Okta and Fleet

Apple's [Platform Single Sign-on (Platform SSO)](https://support.apple.com/guide/deployment/platform-sso-for-macos-dep7bbb05313/web), enables the following features for macOS hosts:
- Initial local account creation based on identity provider (IdP) credentials during macOS automatic (ADE) enrollment (aka [Simplified Platform SSO](#simplified-platform-sso-macos-26))
- Sync local account password with IdP
- End users sign in to their Mac once and automatically access apps and websites that require authentication through an IdP

This guide details how to enable these features by deploying Okta's macOS Platform SSO extension (Desktop Password Sync) to your Fleet macOS hosts.

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

Okta supports two SCEP challenge types: **dynamic** and **static**. When using the dynamic option with Fleet as a SCEP proxy, Fleet automatically renews certificates 30 days before expiration (or at half the validity period if ≤30 days) when `$FLEET_VAR_SCEP_RENEWAL_ID` is included in the OU field of your certificate profile. 

Static challenges require manual redeployment before expiry. See [Okta's Device Access certificates documentation](https://help.okta.com/oie/en-us/content/topics/oda/oda-as-scep.htm) for a full overview.

The recommended approach is to use Fleet as a SCEP proxy with Okta's dynamic challenge. Fleet fetches a unique, short-lived challenge from Okta for each host at enrollment, so no static secret is shared across devices or embedded in your profile. See [Okta's guide to configuring Okta as a CA with a dynamic SCEP challenge](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm) for more details on how dynamic challenges work.

### Option 1: Dynamic SCEP challenge via Fleet (Recommended)

#### Step 1: Generate your Okta SCEP credentials

1. In the Okta Admin Console, go to **Security** → **Device integrations**
2. Click the **Device Access** tab (not Endpoint management)
3. Click **Add platform**
4. Select **Desktop (Windows and macOS only)**, then click **Next**
5. On the Add device management platform page, select:
   - **Certificate authority:** Use Okta as certificate authority
   - **SCEP URL challenge type:** Dynamic SCEP URL
6. Click **Generate**
7. Copy and save the **SCEP URL**, **Challenge URL**, **Username**, and **Password** — you'll need all four for Fleet

#### Step 2: Add Okta as a CA in Fleet

In Fleet, go to **Settings** → **Integrations** → **Certificate authorities** and click **Add CA**. Select **Okta CA or Microsoft Device Enrollment service (NDES)** and enter the values from step 7:

- **SCEP URL:** The SCEP URL from Okta
- **Admin URL:** The Challenge URL from Okta
- **Username** and **Password:** The credentials from Okta

Alternatively, configure via GitOps in your `org_settings`:

```yaml
org_settings:
  integrations:
    ...
  certificate_authorities:
    ndes_scep_proxy:
      url: https://your-okta-org.okta.com/scep
      admin_url: https://your-okta-org.okta.com/scep/challenge
      username: your-username
      password: your-password
```

#### Step 3: Create the SCEP certificate profile

Open [iMazing Profile Editor](https://imazing.com/profile-editor), create a new profile, and add a **SCEP** payload:

**Under the General tab:**
- **Name:** Okta Device Access SCEP
- **Identifier:** Enter a unique string (e.g. `com.okta.device.access.53D4F816-6B96-400A-81A4-2C141E582D54`)
- **UUID:** Make sure this field is populated

**Under SCEP:**
- **URL:** `$FLEET_VAR_NDES_SCEP_PROXY_URL`
- **Challenge:** `$FLEET_VAR_NDES_SCEP_CHALLENGE`
- **Subject:** `CN=managementAttestation %HardwareUUID%`
- **Subject Alt Names:** Add an OU field with value `$FLEET_VAR_SCEP_RENEWAL_ID`
- **Key Size:** 2048
- **Key Usage:** Signing
- **Key is Extractable:** Unchecked
- **Allow All Apps Access:** Checked
- **Certificate Expiration Notification:** Set to 30 days before expiration

**Important:** The Subject must include both the CN and an OU field with `$FLEET_VAR_SCEP_RENEWAL_ID`. In raw XML, the Subject array should look like this:

```xml
<key>Subject</key>
<array>
    <array>
        <array>
            <string>CN</string>
            <string>managementAttestation %HardwareUUID%</string>
        </array>
        <array>
            <string>OU</string>
            <string>$FLEET_VAR_SCEP_RENEWAL_ID</string>
        </array>
    </array>
</array>
```

Fleet replaces `$FLEET_VAR_NDES_SCEP_PROXY_URL`, `$FLEET_VAR_NDES_SCEP_CHALLENGE`, and `$FLEET_VAR_SCEP_RENEWAL_ID` with the appropriate values each time the profile is delivered to a host. Each host receives a unique, short-lived challenge rather than a shared static secret.

> **Important:** Fleet automatically renews this certificate when `$FLEET_VAR_SCEP_RENEWAL_ID` is in the OU field (already included above). Use the osquery policy below to monitor certificate expiry across your fleet.

```sql
-- Returns 1 if all Okta certs are valid for >30 days (PASSING)
-- Returns 0 if any Okta certs expire within 30 days (FAILING)
SELECT 1
WHERE NOT EXISTS (
  SELECT 1
  FROM certificates
  WHERE issuer LIKE '%/DC=com/DC=okta%'
    AND CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) <= 30
    AND CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) >= 0
);
```

Save this as `okta-device-access-scep.mobileconfig`.

**[View example dynamic SCEP profile →](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-device-access-scep-dynamic-example.mobileconfig)**

---

### Option 2: Static SCEP challenge

If you prefer to use a static challenge without Fleet acting as a SCEP proxy, follow these steps instead. See [Okta's guide to configuring Okta as a CA with a static SCEP challenge](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-static-scep-macos-jamf.htm) for more details.

#### Step 1: Generate SCEP URL and secret key

1. In the Okta Admin Console, go to **Security** → **Device integrations**
2. Click the **Device Access** tab (not Endpoint management)
3. Click **Add platform**
4. Select **Desktop (Windows and macOS only)**, then click **Next**
5. On the Add device management platform page, select:
   - **Certificate authority:** Use Okta as certificate authority
   - **SCEP URL challenge type:** Static SCEP URL
6. Click **Generate**
7. Copy and save the **SCEP URL** and **secret key** — you'll need these for your profile

#### Step 2: Create the SCEP certificate profile

On your Mac, open [iMazing Profile Editor](https://imazing.com/profile-editor). Create a new profile and add a **SCEP** payload:

**Under the General tab:**
- **Name:** Okta Device Access SCEP
- **Identifier:** Enter a unique string (e.g. `com.okta.device.access.53D4F816-6B96-400A-81A4-2C141E582D54`)
- **UUID:** Make sure this field is populated

**Under SCEP:**
- **URL:** The SCEP URL from Okta (step 7 above)
- **Challenge:** The secret key from Okta (step 7 above)
- **Subject:** `CN=managementAttestation %HardwareUUID%`
- **Key Size:** 2048
- **Key Usage:** Signing
- **Key is Extractable:** Unchecked
- **Allow All Apps Access:** Checked
- **Certificate Expiration Notification:** Set to 14 days before expiration

> **Important:** Static SCEP challenges require manual redeployment — Fleet's automatic renewal via `$FLEET_VAR_SCEP_RENEWAL_ID` only works when Fleet is acting as a SCEP proxy (dynamic option). Use the osquery policy below to identify hosts with certificates expiring within 14 days.

```sql
SELECT 1
FROM certificates
WHERE issuer LIKE '%/DC=com/DC=okta%'
  AND ca=0
  AND CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) >= 14;
```

**[View example static SCEP profile →](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-device-access-scep-example.mobileconfig)**

---

## Install Okta Verify via Fleet

On your Fleet server, select the fleet you want to deploy Platform SSO to. Navigate to **Software** → **Add software** → **Custom package** → **Choose file**.

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

**[View example Associated Domains profile →](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-associated-domains-example.mobileconfig)**

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

**[View example SSO Extension profile →](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-sso-extension-example.mobileconfig)**

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

**[View example App Configuration profile →](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-app-config-example.mobileconfig)**

**Note:** These example profiles demonstrate the essential configuration options. For a complete reference of all available settings and options, see [Okta's official configuration profile documentation](https://help.okta.com/oie/en-us/content/topics/oda/macos-pw-sync/configure-macos-password-sync-policies.htm).

### 4. Security Preference Profile (Optional)

To prevent users from changing their local password (since it syncs with Okta), create a security preference profile:

- **Preference Domain:** `com.apple.preference.security`
- **Settings:**
  - **dontAllowPasswordResetUI:** `true`

Save as `okta-security-restrictions.mobileconfig`.

## Deploy Configuration Profiles via Fleet

Now deploy all the configuration profiles to your Fleet hosts:

1. On your Fleet server, select the fleet you want to deploy Platform SSO to
2. Navigate to **Controls > OS Settings > Configuration profiles**
3. Upload each profile in this order:
   - `okta-device-access-scep.mobileconfig` (macOS 14+ only)
   - `okta-associated-domains.mobileconfig`
   - `okta-sso-extension.mobileconfig`
   - `okta-app-config.mobileconfig`
   - `okta-security-restrictions.mobileconfig` (optional)

Uploading the profiles to a fleet will automatically deliver them to all macOS hosts enrolled in that fleet. If you wish to have more control over which hosts receive the profiles, you can use labels to target or exclude specific hosts.

**Important:** For organizations with both macOS 13 and macOS 14+ devices, you'll need to create separate fleets or use labels to deploy the appropriate profile versions to each macOS version.

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

## Simplified Platform SSO (macOS 26+)

Apple introduced Simplified Platform SSO in macOS 26. It streamlines the Platform SSO setup by presenting a **Single Sign-On for Mac** page during Setup Assistant, allowing users to authenticate with their IdP right out of the box with no post-enrollment registration step required.

### Prerequisites (Simplified Platform SSO)

In addition to the [standard prerequisites](#prerequisites) above, Simplified Platform SSO requires:

- Hosts running **macOS 26** or later
- Hosts enrolled via **Apple Business (AB)**
- Fleet's **Setup experience** configured for the target fleet
- The latest **Okta Verify** installer downloaded from your Okta tenant or via Fleet-maintained apps (not the App Store version)

### Step 1: Configure profiles

Simplified Platform SSO uses the same Extensible SSO / Platform SSO profile and Okta Device Access SCEP profile described in the sections above. Follow the existing instructions to create:

- An **Extensible Single Sign-On** profile with Simplified Platform SSO settings.

For users to be created during setup and immediately registered with Platform SSO the profile must include **EnableRegistrationDuringSetup** and must list https://yourdomain.okta.com/v1/auth/device-sign within **URLs**

Example configuration for macOS 26:

```xml
<key>PayloadType</key>
<string>com.apple.extensiblesso</string>
<key>PlatformSSO</key>
<dict>
    <key>AccountDisplayName</key>
    <string>Okta Verify</string>
    <key>AuthenticationMethod</key>
    <string>Password</string>
    <key>UseSharedDeviceKeys</key>
    <true/>
    <key>EnableRegistrationDuringSetup</key>
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
    <string>https://yourdomain.okta.com/v1/auth/device-sign</string>
</array>
```
  - View example **[Extensible Single Sign-On profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-sso-extension-simplified-setup-example.mobileconfig)**
- An **Associated domains** profile.
  - View example **[Associated domains profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-associated-domains-example.mobileconfig)**
- An **Okta App configuration** profile. This profile includes IdP variables, if you don't have IdP authentication enabled for enrollment you can delete the key and value for `OktaVerify.UserPrincipalName`.
  - View example **[Okta App configuration profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-app-config-example.mobileconfig)**
- An **Okta Device Access SCEP** certificate profile.
  - View example **[dynamic Okta Device Access SCEP profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-device-access-scep-dynamic-example.mobileconfig)**
  - View example **[static Okta Device Access SCEP profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/okta-device-access-scep-example.mobileconfig)**
 
> Extensible Single Sign-On, Associated domains, and Okta App configuration profiles can be combined into a single profile for simplicity. 
  
Upload all profiles to the target fleet in Fleet under **Controls > OS Settings > Configuration profiles**.
For best results, don't use labels to scope Platform SSO profiles to ensure they're immediately applicable to hosts during setup.

### Step 2: Add Okta Verify as a setup experience app

Download the latest `OktaVerify-Installer.pkg` from your Fleet-maintined apps or Okta Admin Console (**Settings > Downloads**). Don't use the App Store version as it lacks the required MDM integration features.

If downloading from Okta Admin Console, in Fleet navigate to the target fleet and go to **Controls > Setup experience > Install software**. Upload the Okta Verify installer so that it is installed on the host during setup experience.

### Step 3: Enroll via AB

Enroll the host through Apple Business. After setup experience completes (profiles are delivered and Okta Verify is installed), the user is presented with a new **Single Sign-On for Mac** page containing an Okta login prompt.

### End user experience (Simplified Platform SSO)

1. **Single Sign-On for Mac screen:** After setup experience, the user sees an Okta login page. They enter their Okta credentials to authenticate.
2. **Create account screen:** After authenticating, the standard macOS create-account screen appears, but all fields are locked (the user can only edit the **password hint**). The local account password is automatically set to match their Okta password.
3. **Okta Verify setup:** Later in Setup Assistant, the user logs into Okta a second time to finalize the Okta Verify registration on the device.

### Password syncing behavior

With Simplified Platform SSO, the user's local macOS password is tied to their Okta password:

- Users **cannot** set a local password that differs from their Okta password.
- If the Okta password is changed via the Okta web UI, the user may not immediately receive a notification to sync. Locking the screen and unlocking with the **new** Okta password triggers the sync and the old password stops working at that point.

### Multiple credential prompts

When Fleet's **IdP authentication** is also enabled, the user enters IdP credentials **three times** during enrollment:

1. MDM enrollment IdP authentication
2. Platform SSO authentication (Single Sign-On for Mac screen)
3. Okta Verify setup

### Managing mixed macOS versions

If your fleet includes hosts running macOS versions older than macOS 26, carefully review Apple's Platform SSO documentation to understand which features are supported on each version. Consider assigning hosts on older macOS versions to a **separate fleet** in Fleet so they receive the standard Platform SSO profiles (described earlier in this guide) rather than the Simplified Platform SSO configuration.

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

### Simplified Platform SSO: Okta Verify factor on a wiped device

If a user's only Okta Verify factor is registered on the host being set up, and that host is wiped and re-enrolled, the final Okta Verify setup step will fail. Okta's logs do not surface a clear error for this scenario.

**Fix:** Before re-enrolling a wiped device, revoke both the host and the user's Okta Verify registrations in the Okta Admin Console.

### Simplified Platform SSO: SCEP or Okta Verify install failure

If the Okta SCEP certificate enrollment fails or the Okta Verify installation fails during setup experience, the user gets stuck at the Single Sign-On for Mac screen with no clean way to proceed. Verify that the SCEP profile is correctly configured and that the Okta Verify package uploaded to Fleet is the latest version from your Okta tenant. Admins can wipe hosts from Fleet to retry the setup.

## Additional Resources

For more detailed information about configuring Okta Desktop Password Sync, see the [official Okta documentation](https://help.okta.com/oie/en-us/content/topics/oda/macos-pw-sync/configure-macos-password-sync.htm).

To see a full list of properites for the Extensible Single Sign-On configuration profile, see the [Apple documentation](https://developer.apple.com/documentation/devicemanagement/extensiblesinglesignon).

To create and customize configuration profiles, download [iMazing Profile Editor](https://imazing.com/profile-editor).

For Device Access SCEP certificate configuration details, see [Use Okta as a CA for Device Access](https://help.okta.com/oie/en-us/content/topics/oda/oda-as-scep-okta-ca.htm) and [Okta's Device Access certificates documentation](https://help.okta.com/oie/en-us/content/topics/oda/oda-as-scep.htm).

[*Get started with Fleet*](https://fleetdm.com/docs/get-started/why-fleet)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="tux234">
<meta name="authorFullName" value="Mitch Francese">
<meta name="publishedOn" value="2026-03-08">
<meta name="articleTitle" value="Deploying Platform SSO with Okta Device Access">
<meta name="description" value="Learn how to use Fleet to deploy the Okta Platform SSO Extension">
