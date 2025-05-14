# Single sign-on (SSO)

Fleet supports [Okta](#okta), [authentik](#authentik), [Google Workspace](#google-workspace), and [Microsoft Active Directory (AD) / Entra ID](https://learn.microsoft.com/en-us/entra/architecture/auth-saml), as well as any other identity provider (IdP) that supports the SAML standard.

To configure SSO, follow steps for your IdP and then complete [Fleet configuration](#fleet-configuration).

> JIT? SAML implementation supports just-in-time (JIT) user provisioning, as well as both IdP-initiated login and service-initiated (SP) login.


## Okta

Create a new SAML app in Okta:

![Example Okta IdP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

If you're configuring [end user authentication](../Using%20Fleet/MDM-macOS-setup-experience.md#end-user-authentication-and-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` for the **Single sign on URL** instead.

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
    - For **ACS URL**, use `https://<your_fleet_url>/api/v1/fleet/sso/callback`. If you're configuring [end user authentication](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` instead.
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
   - For **Reply URL (Assertion Consumer Service URL)**, enter `https://<your_fleet_url>/api/v1/fleet/sso/callback`. If you're configuring [end user authentication](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula), use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` instead.
   - Click **Save**.
6. In the **(3) SAML Certificates** box, click the copy button in the **App Federation Metadata Url** field.
 ![The new SAML app's details page in Enta Admin Center](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/entra-sso-configuration-step-6.png)

On your Fleet server: 
1. Navigate to **Settings > Organization settings > Single sign-on options**.
2. On the **Single sign-on options** page:
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

2. Navigate to **Applications -> Applications** and click **Create with Provider** to create an application and provider pair.

3. Enter "Fleet" for the **App name** and click **Next**.

4. Choose **SAML** as the **Provider Type** and click **Next**.
    - For **Name**, enter "Fleet".
    - For **Authorization flow**, choose `default-provider-authorization-implicit-consent (Authorize Application)`.
    - In the **Protocol settings** section, configure the following:
      - For **Assertion Consumer Service URL** use `https://<your_fleet_url>/api/v1/fleet/sso/callback`.
        - If you're configuring **[end user authentication](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula)**, use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback`.
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

This section explains how JIT user provisioning works. With JIT, Fleet will automatically create a user account when someone logs in for the first time using your configured SSO. This removes the need to create individual user accounts for a large organization.

When JIT user provisioning is turned on, Fleet will automatically create an account when a user logs in for the first time with the configured SSO.

The new account's email and full name are copied from the user data in the SSO response.
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
