Configuring Single Sign On
===========================

Kolide supports SAML single sign on capability.  This feature is convenient for users and offloads responsibility for user authentication to a third party identity provider such as
Salesforce or Onelogin.  Kolide supports the SAML Web Browser SSO Profile using the HTTP Redirect Binding.  

## Identity Provider (IDP) Configuration

Several items are required to configure an IDP to provide SSO services to Kolide. Note that the names of these items may vary from provider to provider and may not conform to the SAML spec. Individual users must also be setup on the IDP before
they can sign in to Kolide.  The particulars of setting up the connected application (Kolide) and users will vary for different identity providers but will generally require the following
information.  

* _Assertion Consumer Service_ - This is the call back URL that the identity provider
will use to send security assertions to Kolide. The value that you supply will be a fully qualified URL
consisting of your Kolide web address and the callback path `/api/v1/kolide/sso/callback`. For example,
if your Kolide web address is https://acme.kolide.com, then the value you would
use in the identity provider configuration would be:

  ```
  https://acme.kolide.com/api/v1/kolide/sso/callback
  ```

* _Entity ID_ - This value is a URI that you define. It identifies your Kolide instance as the service provider that issues authorization requests. The value must exactly match the
Entity ID that you define in the Kolide SSO configuration.

* _Name ID Format_ - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IDP setup to `email` or similar.

* _Subject Type_ - `username`.

  #### Example IDP Configuration  

  ![Example IDP Configuration](../images/idp-setup.png)

The IDP will generate an issuer URI and a metadata URL that will be used to configure
Kolide as a service provider.

## Kolide SSO Configuration

A user must be an admin to configure Kolide for SSO.  The SSO configuration is
found in App Settings. The following values are required.

* _Identity Provider Name_ - A human friendly name of the IDP.

* _Entity ID_ - A URI that identifies your Kolide instance as the issuer of authorization
requests. Assuming your company name is Acme, an example might be `acme.kolide.com` although
the value could be anything as long as it is unique to Kolide as a service provider
and matches the entity provider value used in the IDP configuration.

* _Issuer URI_ - This value is obtained from the IDP.

* _Metadata URL_ - This value is obtained from the IDP and is used by Kolide to
issue authorization requests to the IDP.

* _Metadata_ - If the IDP does not provide a metadata URL, the metadata must
be obtained from the IDP and entered. Note that the metadata URL is preferred if
the IDP provides metadata in both forms.

### Example Kolide SSO Configuration

![Example SSO Configuration](../images/sso-setup.png)

## Creating SSO Users in Kolide

When an admin invites a new user to Kolide, they may select the `Enable SSO` option. The
SSO enabled users will not be able to sign in with a regular user ID and password.  It is
strongly recommended that at least one admin user is set up to use the traditional password
based log in so that there is a 'back door' to log into Kolide and modify the SSO
configuration in the event of problems.   




[SAML Bindings](http://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf)

[SAML Profiles](http://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf)
