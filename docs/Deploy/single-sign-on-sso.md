# Single sign-on (SSO)

Learn how to configure single sign-on (SSO) and just-in-time (JIT) user provisioning.

## Overview

Fleet supports SAML single sign-on capability.

Fleet supports both SP-initiated SAML login and IDP-initiated login. However, IDP-initiated login must be enabled in the web interface's SAML single sign-on options.

Fleet supports the SAML Web Browser SSO Profile using the HTTP Redirect Binding.

> Note: The email used in the SAML Assertion must match a user that already exists in Fleet unless you enable [JIT provisioning](#just-in-time-jit-user-provisioning).**

## Identity provider (IDP) configuration

Setting up the service provider (Fleet) with an identity provider generally requires the following information:

- _Assertion Consumer Service_ - This is the call-back URL that the identity provider
  will use to send security assertions to Fleet. In Okta, this field is called _single sign-on URL_. On Google, it is "ACS URL." The value you supply will be a fully qualified URL consisting of your Fleet web address and the call-back path `/api/v1/fleet/sso/callback`. For example, if your Fleet web address is https://fleet.example.com, then the value you would use in the identity provider configuration would be:
  ```
  https://fleet.example.com/api/v1/fleet/sso/callback
  ```

- _Entity ID_ - This value is an identifier that you choose. It identifies your Fleet instance as the service provider that issues authorization requests. The value must match the Entity ID that you define in the Fleet SSO configuration.

- _Name ID Format_ - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IDP setup to something like `email` or `EmailAddress`.

- _Subject Type (Application username in Okta)_ - `email`.

After supplying the above information, the IDP will generate an issuer URI and metadata that will be used to configure Fleet as a service provider.

## Fleet SSO configuration

A Fleet user must be assigned the Admin role to configure Fleet for SSO. In Fleet, SSO configuration settings are located in **Settings > Organization settings > SAML single sign-on options**.

If your IDP supports dynamic configuration, like Okta, you only need to provide an _identity provider name_ and _entity ID_, then paste a link in the metadata URL field. Make sure you create the SSO application within your IDP before configuring it in Fleet.

Otherwise, the following values are required:

- _Identity provider name_ - A human-readable name of the IDP. This is rendered on the login page.

- _Entity ID_ - A URI that identifies your Fleet instance as the issuer of authorization
  requests (e.g., `fleet.example.com`). This must match the _Entity ID_ configured with the IDP.

- _Metadata URL_ - Obtain this value from the IDP and is used by Fleet to
  issue authorization requests to the IDP.

- _Metadata_ - If the IDP does not provide a metadata URL, the metadata must
  be obtained from the IDP and entered. Note that the metadata URL is preferred if
  the IDP provides metadata in both forms.

### Example Fleet SSO configuration

![Example SSO Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/sso-setup.png)

## Creating SSO users in Fleet

When an admin creates a new user in Fleet, they may select the `Enable single sign on` option. The
SSO-enabled users will not be able to sign in with a regular user ID and password.

It is strongly recommended that at least one admin user is set up to use the traditional password-based login so that there is a fallback method for logging into Fleet in the event of SSO
configuration problems.

> Individual users must also be set up on the IDP before signing in to Fleet.

## Enabling SSO for existing users in Fleet
As an admin, you can enable SSO for existing users in Fleet. To do this, go to the Settings page,
then click on the Users tab. Locate the user you want to enable SSO for, and in the Actions dropdown
menu for that user, click on "Edit." In the dialogue that opens, check the box labeled "Enable
single sign-on," then click "Save." If you are unable to check that box, you must first [configure
and enable SSO for the organization](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

## Just-in-time (JIT) user provisioning

`Applies only to Fleet Premium`

When JIT user provisioning is turned on, Fleet will automatically create an account when a user logs in for the first time with the configured SSO. This removes the need to create individual user accounts for a large organization.

The new account's email and full name are copied from the user data in the SSO response.
By default, accounts created via JIT provisioning are assigned the [Global Observer role](https://fleetdm.com/docs/using-fleet/permissions).
To assign different roles for accounts created via JIT provisioning see [Customization of user roles](#customization-of-user-roles) below.

To enable this option, go to **Settings > Organization settings > single sign-on options** and check "_Create user and sync permissions on login_" or [adjust your config](#sso-settings-enable-jit-provisioning).

For this to work correctly make sure that:

- Your IDP is configured to send the user email as the Name ID (instructions for configuring different providers are detailed below)
- Your IDP sends the full name of the user as an attribute with any of the following names (if this value is not provided Fleet will fallback to the user email)
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

### Okta IDP configuration

![Example Okta IDP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

Once configured, you will need to retrieve the Issuer URI from the `View Setup Instructions` and metadata URL from the `Identity Provider metadata` link within the application `Sign on` settings. See below for where to find them:

![Where to find SSO links for Fleet](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-retrieve-links.png)

> The Provider Sign-on URL within the `View Setup Instructions` has a similar format as the Provider SAML Metadata URL, but this link provides a redirect to _sign into_ the application, not the metadata necessary for dynamic configuration.

> The names of the items required to configure an identity provider may vary from provider to provider and may not conform to the SAML spec.

### Google Workspace IDP Configuration

Follow these steps to configure Fleet SSO with Google Workspace. This will require administrator permissions in Google Workspace.

1. Navigate to the [Web and Mobile Apps](https://admin.google.com/ac/apps/unified) section of the Google Workspace dashboard. Click _Add App -> Add custom SAML app_.

  ![The Google Workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-1.png)

2. Enter `Fleet` for the _App name_ and click _Continue_.

  ![Adding a new app to Google workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-2.png)

3. Click _Download Metadata_, saving the metadata to your computer. Click _Continue_.

  ![Download metadata](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-3.png)

4. In Fleet, navigate to the _Organization Settings_ page. Configure the _SAML single sign-on options_ section.

  - Check the _Enable single sign-on_ checkbox.
  - For _Identity provider name_, use `Google`.
  - For _Entity ID_, use a unique identifier such as `fleet.example.com`. Note that Google seems to error when the provided ID includes `https://`.
  - For _Metadata_, paste the contents of the downloaded metadata XML from step three.
  - All other fields can be left blank.

  Click _Update settings_ at the bottom of the page.

  ![Fleet's SAML single sign on options page](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-4.png)

5. In Google Workspace, configure the _Service provider details_.

  - For _ACS URL_, use `https://<your_fleet_url>/api/v1/fleet/sso/callback` (e.g., `https://fleet.example.com/api/v1/fleet/sso/callback`).
  - For Entity ID, use **the same unique identifier from step four** (e.g., `fleet.example.com`).
  - For _Name ID format_, choose `EMAIL`.
  - For _Name ID_, choose `Basic Information > Primary email`.
  - All other fields can be left blank.

  Click _Continue_ at the bottom of the page.

  ![Configuring the service provider details in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-5.png)

6. Click _Finish_.

  ![Finish configuring the new SAML app in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-6.png)

7. Click the down arrow on the _User access_ section of the app details page.

  ![The new SAML app's details page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-7.png)

8. Check _ON for everyone_. Click _Save_.

  ![The new SAML app's service status page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-8.png)

9. Enable SSO for a test user and try logging in. Note that Google sometimes takes a long time to propagate the SSO configuration, and it can help to try logging in to Fleet with an Incognito/Private window in the browser.

<meta name="title" value="Single sign-on (SSO)">
<meta name="pageOrderInSection" value="800">
<meta name="description" value="Learn how to configure single sign-on (SSO)">
<meta name="navSection" value="TBD">