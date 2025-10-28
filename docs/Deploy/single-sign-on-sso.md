# Single sign-on (SSO)

Fleet supports [Okta](#okta), [authentik](#authentik), [Google Workspace](#google-workspace), and [Microsoft Active Directory (AD) / Entra ID](https://learn.microsoft.com/en-us/entra/architecture/auth-saml), as well as any other identity provider (IdP) that supports the SAML standard.

To configure SSO, follow steps for your IdP and then complete [Fleet configuration](#fleet-configuration).

> JIT? SAML implementation supports just-in-time (JIT) user provisioning, as well as both IdP-initiated login and service-initiated (SP) login.

**Using Fleet MDM?** If you're using automatic enrollment (ADE/DEP), you'll need two separate SSO apps in your IdP - one for your admin console and one for end user authentication during device setup. See [End user authentication for MDM](#end-user-authentication-for-mdm) after you've configured your IdP.


## Okta

Fleet offers two ways to set up Okta:

1. **[Okta Integration Network (OIN)](#okta-integration-network-oin)** - Use the pre-configured Fleet app from Okta's catalog (recommended for most users)
2. **[Custom SAML app](#okta-custom-saml-app)** - Manually create a SAML app in Okta (for testing or advanced configurations)

### Okta Integration Network (OIN)

The Fleet app is available in the Okta Integration Network catalog. This is the fastest way to set up SAML SSO and SCIM provisioning for your Fleet admin console.

**Note:** The OIN app is for Fleet admin/user access only. If you're using MDM with automatic enrollment, you'll need a separate custom SAML app for end user authentication. See [End user authentication for MDM](#end-user-authentication-for-mdm) below.

**What you'll need:**
- Fleet Premium license (required for both SAML SSO and SCIM provisioning)
- Fleet admin access
- Okta admin access

#### Supported features

**SAML 2.0 Single Sign-On:**
- SP-initiated SSO
- IdP-initiated SSO
- Just-In-Time (JiT) provisioning

**SCIM 2.0 Provisioning:**
- Create users
- Update user attributes
- Deactivate users
- Reactivate users
- Push groups

#### Add Fleet from the OIN catalog

1. Sign in to your Okta Admin Console as an administrator
2. Go to **Applications** > **Applications**
3. Click **Browse App Catalog**
4. Search for "Fleet" and select the Fleet application
5. Click **Add Integration**
6. On the General Settings page, configure the application label (optional) and click **Done**

#### Configure SAML settings

1. In the Fleet application, go to the **General** tab
2. Click **Edit** in the App Settings section
3. Configure:
   - **Application label**: Fleet (or whatever you want users to see)
   - **Entity ID**: Choose any unique identifier (e.g., `fleet` or `fleet.example.com`) - you'll use this same value in Fleet later
   - **Fleet Instance Base URL**: Your Fleet server domain without https:// (e.g., `acmeco.cloud.fleetdm.com`)
4. Click **Save**

#### Assign users to Fleet

1. Go to the **Assignments** tab
2. Click **Assign** and choose:
   - **Assign to People**: For individual users
   - **Assign to Groups**: For groups of users
3. Click **Assign** next to the users or groups you want
4. Click **Done**

#### Complete SAML configuration in Fleet

After configuring SAML in Okta, finish the setup in Fleet:

1. In the Fleet application in Okta, go to the **Sign On** tab
2. Under **SAML 2.0** > **Metadata details**, copy the Metadata URL
3. In Fleet, go to **Settings** > **Integrations** > **Single sign-on (SSO)**
4. Configure:
   - Check **Enable single sign-on**
   - **Identity provider name**: Okta (or whatever you want to call it)
   - **Entity ID**: Use the **exact same value** you entered in Okta (e.g., `fleet`)
   - **Metadata URL**: Paste the URL from step 2
5. Click **Save**

**Important:** The Entity ID must match exactly between Okta and Fleet or SSO won't work.

Once SAML is set up, users can sign into Fleet directly from Fleet's login page:

1. Go to your Fleet instance login page
2. Click the **Login with Okta** button below the email/password fields
3. Enter your Okta credentials
4. You'll be redirected back to Fleet and signed in

This means users can bookmark your Fleet instance and always start there, rather than going through the Okta dashboard.

#### Configure SCIM provisioning

SCIM automates user and group management between Okta and Fleet.

##### Create Fleet API token

First, create an API token in Fleet:

1. Install fleetctl (Fleet's command-line tool)
2. Run this command to create an API-only user with maintainer permissions:
   ```
   fleetctl user create --name 'API User' --email 'api@example.com' --password 'temp@pass123' --api-only --global-role 'maintainer'
   ```
3. Copy the API token from the output

For detailed instructions, see [Create API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user).

##### Enable provisioning in Okta

1. In the Fleet application, go to the **Provisioning** tab
2. Click **Configure API Integration**
3. Check **Enable API integration**
4. Enter:
   - **Base URL**: `https://your-fleet-instance.com/api/v1/fleet/scim`
   - **API Token**: Paste your Fleet API token
5. Click **Test API Credentials** to verify the connection
6. If successful, click **Save**

##### Configure provisioning settings

1. Under the **Provisioning** tab, click **To App** in the left sidebar
2. Click **Edit** in the Provisioning to App section
3. Enable:
   - **Create Users**: Automatically create users in Fleet when assigned in Okta
   - **Update User Attributes**: Update user info when changed in Okta
   - **Deactivate Users**: Deactivate users in Fleet when unassigned or deactivated in Okta
4. Click **Save**

##### Configure attribute mappings

1. On the **Provisioning** tab, click **To App**
2. Scroll to the attribute mappings section
3. Make sure these required attributes are mapped:
   - **userName** → Okta username
   - **givenName** → First name
   - **familyName** → Last name
4. Optional attributes:
   - **department** → Department
   - **FLEET_JIT_USER_ROLE_GLOBAL** → User's Fleet role (see below)
5. Remove any other unmapped attributes that Fleet doesn't support
6. Click **Save**

**Setting up FLEET_JIT_USER_ROLE_GLOBAL:**

This attribute automatically assigns Fleet permissions when users are provisioned. It's especially useful with Just-In-Time (JiT) provisioning.

Supported values:
- `admin` - Full administrative access to Fleet
- `maintainer` - Can manage hosts, queries, policies, and software
- `observer_plus` - Read-only access plus ability to run queries
- `observer` - Read-only access to Fleet

How to configure:
1. In the attribute mappings, find **FLEET_JIT_USER_ROLE_GLOBAL**
2. Map it to an Okta user attribute with the desired Fleet role
3. Common setups:
   - Static value: Set to `"observer"` (with quotes) for all users
   - Group-based: Use an expression like `String.contains(user.groups, "Fleet Admins") ? "admin" : "observer"`

If you don't configure this, provisioned users default to observer role.

##### Push groups (optional)

To sync Okta groups with Fleet:

1. Go to the **Push Groups** tab
2. Click **Push Groups** > **Find groups by name**
3. Search for and select the groups you want to push
4. Make sure **Push group memberships immediately** is selected
5. Click **Save**

#### Verify everything works

1. Sign in to Fleet as an admin
2. Go to **Settings** > **Integrations** > **Identity provider (IdP)**
3. Check that Fleet's receiving requests from Okta
4. Verify users and groups are showing up correctly

#### Troubleshooting

**Users not provisioning:**
- Check that your SCIM API token is valid and has maintainer permissions
- Look for error messages in the **Provisioning** tab in Okta
- Verify all required attributes (userName, givenName, familyName) are mapped
- Double-check your Base URL: `https://your-fleet-instance.com/api/v1/fleet/scim`

**SAML authentication issues:**
- Make sure your Fleet Server URL doesn't have a trailing slash
- Confirm users are assigned to the Fleet app in Okta
- If you're not using SCIM, check that Just-In-Time provisioning is enabled
- Verify the SAML metadata is configured in Fleet

**Group membership not syncing:**
- Confirm groups are pushed in the **Push Groups** tab
- Check that **Push group memberships immediately** is enabled
- Verify users are actually members of the pushed groups in Okta

**Need help?**
- Work with your Fleet customer success manager
- Email Fleet support: support@fleetdm.com
- Check Fleet docs: https://fleetdm.com/docs

### Okta custom SAML app

Create a new SAML app in Okta:

![Example Okta IdP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

If you're configuring [end user authentication](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` for the **Single sign on URL** instead.

Once configured, you will need to retrieve the issuer URI from **View Setup Instructions** and metadata URL from the **Identity Provider metadata** link within the application **Sign on** settings. See below for where to find them:

![Where to find SSO links for Fleet](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-retrieve-links.png)

> The Provider Sign-on URL within **View Setup Instructions** has a similar format as the Provider SAML Metadata URL, but this link provides a redirect to _sign into_ the application, not the metadata necessary for dynamic configuration.

## Google Workspace

Create a new SAML app in Google Workspace:

1. Navigate to the [Web and Mobile Apps](https://admin.google.com/ac/apps/unified) section of the Google Workspace dashboard. Click **Add App -> Add custom SAML app**.

  ![The Google Workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-1.png)

2. Enter "Fleet" for the **App name** and click **Continue**.

  ![Adding a new app to Google Workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-2.png)

3. Click **Download Metadata**, saving the metadata to your computer. Click **Continue**.

  ![Download metadata](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-3.png)

4. Configure the **Service provider details**:
    - For **ACS URL**, use `https://<your_fleet_url>/api/v1/fleet/sso/callback`. If you're configuring [end user authentication](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` instead.
    - For Entity ID, use **the same unique identifier from step four** (e.g., "fleet.example.com").
    - For **Name ID format**, choose `EMAIL`.
    - For **Name ID**, choose `Basic Information > Primary email`.
    - All other fields can be left blank.

  Click **Continue** at the bottom of the page.

  ![Configuring the service provider details in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-5.png)

5. Click **Finish**.

  ![Finish configuring the new SAML app in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-6.png)

6. Click the down arrow on the **User access** section of the app details page.

  ![The new SAML app's details page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-7.png)

7. Check **ON for everyone**. Click **Save**.

  ![The new SAML app's service status page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-8.png)

8. Enable SSO for a test user and try logging in. Note that Google sometimes takes a long time to propagate the SSO configuration, and it can help to try logging in to Fleet with an Incognito/Private window in the browser.

## Entra
Create a new SAML app in Microsoft Entra Admin Center:
1. From the left sidebar, navigate to **Applications > Enterprise Applications**.
2. At the top of the page, click **+ New Application**.
3. On the next page, click **+ Create your own application** and enter the following.
   - For **Input name**, enter `Fleet`.
   - For **What are you looking to do with your application?**, select `Integrate any other application you don't find in the gallery (Non-gallery)`.
   - Click **Create**.
4. In your newly crated Fleet app, select **Single sign-on** from the menu on the left. Then, on the Single sign-on page, select **SAML**.
5. Click the **Edit** button in the (1) Basic SAML Configuration Box.
   - For **Identifier (Entity ID)**, click **Add identifier** and enter `fleet`.
   - For **Reply URL (Assertion Consumer Service URL)**, enter `https://<your_fleet_url>/api/v1/fleet/sso/callback`. If you're configuring [end user authentication](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` instead.
   - Click **Save**.
6. In the **(3) SAML Certificates** box, click the copy button in the **App Federation Metadata Url** field.
 ![The new SAML app's details page in Enta Admin Center](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/entra-sso-configuration-step-6.png)

On your Fleet server: 
1. Navigate to **Settings > Organization settings > Single sign-on (SSO)**.
2. On the **Single sign-on (SSO)** page:
   - Check the box to **Enable single sign-on**.
   - For **Identity provider name**, enter `Entra`.
   - For **Entity ID**, enter `fleet`.
   - In the **Metadata URL** field, paste the URL that you copied from Entra in step 6 in the previous section.
   - Click **Save**.

 ![The configuration for the SSO connection in Fleet](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/entra-sso-configuration-fleet-config.png) 
3. Enable SSO for a test user and try to log in with Entra.
   


## authentik

Fleet can be configured to use authentik as an identity provider. To continue, you will need to have an authentik instance hosted on an HTTPS domain, and an admin account.


1. Log in to authentik and click **Admin interface**.

2. Navigate to **Applications -> Applications** and click **Create with Provider** to create an application and provider pair.

3. Enter "Fleet" for the **App name** and click **Next**.

4. Choose **SAML** as the **Provider Type** and click **Next**.
    - For **Name**, enter "Fleet".
    - For **Authorization flow**, choose `default-provider-authorization-implicit-consent (Authorize Application)`.
    - In the **Protocol settings** section, configure the following:
      - For **Assertion Consumer Service URL** use `https://<your_fleet_url>/api/v1/fleet/sso/callback`.
        - If you're configuring **[end user authentication](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula)**, use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback`.
      - For **Issuer**, use `authentik`.
      - For **Service Provider Binding**, choose `Post`.
      - For **audience**, use `https://<your_fleet_url>`.
    - In the **Advanced protocol settings** section, configure the following:
      - Choose a signing certificate and enable **Sign assertions** and **Sign responses**.
      - For **NameID Property Mapping**, choose `default SAML Mapping: Email`.
    - Click **Next**.
    - Continue to the **Review and Submit Application** page and click **Submit**.

5. Navigate to **Applications -> Providers** and click on the Fleet provider you just created.
    - In the **Related objects** section, click **Copy Metadata URL** and paste the URL to a text editor for later use.

6. Proceed to [Fleet configuration](#fleet-configuration).

## Other IdPs

IdPs generally requires the following information:

- Assertion Consumer Service - This is the call-back URL that the identity provider will use to send security assertions to Fleet. Use `https://<your_fleet_url>/api/v1/fleet/sso/callback`. If you're configuring end user authentication, use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` instead.

- Entity ID - This value is an identifier that you choose. It identifies your Fleet instance as the service provider that issues authorization requests. The value must match the Entity ID that you define in the Fleet SSO configuration.

- Name ID Format - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IdP setup to something like `email` or `EmailAddress`.

- Subject Type - `email`.

After supplying the above information, your IdP will generate an issuer URI and metadata that will be used to configure Fleet as a service provider.

## End user authentication for MDM

If you're using Fleet MDM with automatic enrollment (ADE/DEP), you need **two separate SSO apps** in your IdP - one for the Fleet admin console and one for end user authentication during device setup.

**Why two apps?**

Having separate apps gives you flexibility with security:
- **Admin console app**: Fleet admins and users sign into Fleet's web UI (`/api/v1/fleet/sso/callback`)
- **End user auth app**: Employees authenticate during out-of-box macOS setup (`/api/v1/fleet/mdm/sso/callback`)

With two apps, you can apply different conditional access policies or security controls. Maybe you want MFA required for admins but not during device setup. Or stricter device compliance checks for the admin portal. Separate apps let you tailor security to each use case.

**Setting up the end user authentication app:**

You'll need to create a custom SAML app for end user authentication. The OIN app doesn't support the MDM callback URL yet.

1. Create a new custom SAML app in your IdP (follow the [Okta custom SAML app](#okta-custom-saml-app) instructions if using Okta)
2. Use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` for the SSO callback URL
3. Set **Name ID** to email (required) - Fleet uses this to populate the macOS account name
4. Assign users who'll be setting up new Macs
5. In Fleet, go to **Settings** > **Integrations** > **Mobile device management (MDM)** > **End user authentication** and configure the connection
6. Enable it at **Controls** > **Setup experience** > **End user authentication**

For complete setup instructions, see the [macOS setup experience guide](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula).

## Fleet configuration

To configure SSO in Fleet head to **Settings > Integrations > Single sign-on (SSO)**.

If you're configuring end user authentication head to **Settings > Integrations > Automatic enrollment > End user authentication**.

- **Identity provider name** - A human-readable name of the IdP. This is rendered on the login page.

- **Entity ID** - A URI that identifies your Fleet instance as the issuer of authorization requests (e.g., `fleet.example.com`). This must match the Entity ID configured with the IdP.

- **Metadata URL** - Obtain this value from your IdP. and is used by Fleet to
  issue authorization requests to the IdP.

- **Metadata** - If the IdP does not provide a metadata URL, the metadata must
  be obtained from the IdP and entered.

![Example SSO Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/sso-setup.png)

## Just-in-time (JIT) user provisioning

`Applies only to Fleet Premium`

Fleet automates user creation using just-in-time (JIT) provisioning. Fleet uses System for Cross-domain Identity Management (SCIM) to [map end users' identity provider (IdP) information to host vitals](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts). SCIM for user provisioning is coming soon.

When JIT user provisioning is turned on, Fleet will automatically create an account when a user logs in for the first time with the configured SSO. This removes the need to create individual user accounts for a large organization. The new account's email and full name are copied from the user data in the SSO response.

By default, accounts created via JIT provisioning are assigned the [Global Observer role](https://fleetdm.com/docs/using-fleet/permissions).
To assign different roles for accounts created via JIT provisioning, see [Customization of user roles](#customization-of-user-roles) below.

To enable this option, go to **Settings > Integrations > Single sign-on (SSO)** and check "_Create user and sync permissions on login_" or [adjust your config](#sso-settings-enable-jit-provisioning).

For this to work correctly make sure that:

- Your IdP is configured to send the user email as the Name ID (instructions for configuring different providers are detailed below)
- Your IdP sends the full name of the user as an attribute with any of the following names (if this value is not provided, Fleet will fall back to the user email)
  - `name`
  - `displayname`
  - `cn`
  - `urn:oid:2.5.4.3`
  - `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`

### Customization of user roles

> **Note:** This feature requires setting `sso_settings.enable_jit_provisioning` to `true`.

Users created via JIT provisioning can be assigned Fleet roles using SAML custom attributes that are sent by the IdP in `SAMLResponse`s during login.
Fleet will attempt to parse SAML custom attributes with the following format:
- `FLEET_JIT_USER_ROLE_GLOBAL`: Specifies the global role to use when creating the user.
- `FLEET_JIT_USER_ROLE_TEAM_<TEAM_ID>`: Specifies team role for team with ID `<TEAM_ID>` to use when creating the user.

Currently supported values for the above attributes are: `admin`, `maintainer`, `observer`, `observer_plus` and `null`.
A role attribute with value `null` will be ignored by Fleet. (This is to support limitations on some IdPs which do not allow you to choose what keys are sent to Fleet when creating a new user.)
SAML supports multi-valued attributes, Fleet will always use the last value.

NOTE: Setting both `FLEET_JIT_USER_ROLE_GLOBAL` and `FLEET_JIT_USER_ROLE_TEAM_<TEAM_ID>` will cause an error during login as Fleet users cannot be Global users and belong to teams.

Following is the behavior that will take place on every SSO login:

If the account does not exist then:
  - If the `SAMLResponse` has any role attributes then those will be used to set the account roles.
  - If the `SAMLResponse` does not have any role attributes set, then Fleet will default to use the `Global Observer` role.

If the account already exists:
  - If the `SAMLResponse` has any role attributes then those will be used to update the account roles.
  - If the `SAMLResponse` does not have any role attributes set, no role change is attempted.

Here's a `SAMLResponse` sample to set the role of SSO users to Global `admin`:

```xml
[...]
<saml2:Assertion ID="id16311976805446352575023709" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
  [...]
  <saml2:Subject xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">bar@foo.example.com</saml2:NameID>
    <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml2:SubjectConfirmationData InResponseTo="id1Juy6Mx2IHYxLwsi" NotOnOrAfter="2023-02-27T17:46:53.506Z" Recipient="https://foo.example.com/api/v1/fleet/sso/callback"/>
    </saml2:SubjectConfirmation>
  </saml2:Subject>
  [...]
  <saml2:AttributeStatement xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_GLOBAL" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">admin</saml2:AttributeValue>
    </saml2:Attribute>
  </saml2:AttributeStatement>
</saml2:Assertion>
[...]
```

Here's a `SAMLResponse` sample to set the role of SSO users to `observer` in team with ID `1` and `maintainer` in team with ID `2`:

```xml
[...]
<saml2:Assertion ID="id16311976805446352575023709" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
  [...]
  <saml2:Subject xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">bar@foo.example.com</saml2:NameID>
    <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml2:SubjectConfirmationData InResponseTo="id1Juy6Mx2IHYxLwsi" NotOnOrAfter="2023-02-27T17:46:53.506Z" Recipient="https://foo.example.com/api/v1/fleet/sso/callback"/>
    </saml2:SubjectConfirmation>
  </saml2:Subject>
  [...]
  <saml2:AttributeStatement xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_1" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">observer</saml2:AttributeValue>
    </saml2:Attribute>
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_2" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">maintainer</saml2:AttributeValue>
    </saml2:Attribute>
  </saml2:AttributeStatement>
</saml2:Assertion>
[...]
```

Each IdP will have its own way of setting these SAML custom attributes, here are instructions for how to set it for Okta: https://support.okta.com/help/s/article/How-to-define-and-configure-a-custom-SAML-attribute-statement?language=en_US.

## Email two-factor authentication (2FA)

If you have a "break glass" Fleet user account that's used to login to Fleet when your identify provider (IdP) goes down, you can enable email 2FA, also known as multi-factor authentication (MFA), for this user. For all other users, the best practice is to enable single-sign on (SSO). Then, you can enforce any 2FA method supported by your IdP (i.e. authenticator app, security key, etc.).

Users with email 2FA enabled will get this email when they login to Fleet:

![Example two-factor authentication (2FA) email](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/email-two-factor-authentication-576x638@2x.png)

You can't edit the authentication method for your currently logged-in user. To enable email 2FA for a user, login with a different user who has the admin role and head to **Settings > Users**.

<meta name="title" value="Single sign-on (SSO)">
<meta name="pageOrderInSection" value="200">
<meta name="description" value="Learn how to configure single sign-on (SSO)">
