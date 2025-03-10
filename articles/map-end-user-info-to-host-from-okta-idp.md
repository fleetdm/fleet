# Map end user's information from identity provider (IdP) to host

_Available in Fleet Premium._

Learn how to connect your identity provider (IdP) to retrieve end-user information and map it to a host. This simplifies the process of identifying which employee is assigned to each host. Fleet uses System for Cross-domain Identity Management (SCIM) protocol to retreive users from IdP. This feature is available in Fleet [v4.67.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.67.0).

Currently, Fleet supports only Okta IdP.

## Connect Okta

To connect Okta to Fleet, follow these steps:

1. Head to Okta admin dashboard
2. Select **Applications > Applications** in the side menu, then select **Create App Integration**.
3. Select **SAML 2.0** from offered options and select **Next**
4. On **General Settings** page, add friendly **App name** (e.g Fleet SCIM), and select **Next**.
5. On **SAML Settings** page, add any URL to **Single sign-on URL** and **Audience URI (SP Entity ID)** fields, and select **Next**.
>Okta requires to setup SAML settings in order to setup SCIM integration. Since we don't need SSO, URL can be anything.
6. On **Feedback** page, provide feedback if you want, and select **Finish**.
7. Select **General** tab of the newly created app, then select **Edit** in **App Settings**.
8. For **Provisioning**, select **SCIM**, and select **Save**.
9. Select **Provisioning** tab, then in **SCIM Connection**, select **Edit**.
10. In **SCIM connector base URL**, enter `https://<your_fleet_server_url>/api/v1/fleet/scim`.
11. In **Unique identifier field for users**, enter `userName`.
12. For **Supported provisioning actions**, select **Push New Users**, **Push Profile Updates**, and **Push Groups**.
13. For **Authentication Mode** select **HTTP Header**.
14. Create Fleet API-only user with maintainer permissions, copy API token for that user, and paste it to Okta, in **Authorization** field.
15. Select **Test Connector Configuration** button. You should see success message in Okta.
16. Head to Fleet, select **Settings > Integrations > Identity provider** and verify that Fleet successfully received the request from IdP.
17. Back in Okta, select **Save**
18. Under **Provisioning** tab, select **To App**, and make sure that `givenName` and `familyName` have Okta value assigned. Currently, Fleet will use `userName`, `givenName`, and `familyName`.


### Assign users and groups to Fleet

To send users and groups information to Fleet, you have to assign them to SCIM app that you created previously.

1. On **Applications**, page select app that you created, then select **Push Groups** tab.
2. Select **+ Push Groups** button, and select one of the options. You can select groups manually, or create rule that will push groups that match that rule. Select **Immediately push groups found by this rule** no matter which option you choose.
3. Once selected, groups will be visible in the list and **Push Status** should be **Active**.
4. Go to **Assignments** tab, ...

