# Add user information from identity provider (IdP) to host

_Available in Fleet Premium._

Learn how to connect your identity provider (IdP) to retrieve end-user information and map it to a
host. Fleet uses [SCIM](https://scim.cloud/) protocol.


## Connect Okta

To connect Okta to Fleet, follow these steps:

1. Head to Okta admin dashboard
2. Select **Applications > Applications** in the main menu, then select **Create App Integration**.
3. Select **SAML 2.0** option and select **Next**.
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
18. Under **Provisioning** tab, select **To App**, then select **Edit** in **Provisioning to App** section. Enable **Create Users**, **Update User Attributes**, and **Deactivate Users**, then select **Save**.
19. On the same page, make sure that `givenName` and `familyName` have Okta value assigned to it.
    Currently, Fleet support `userName`, `givenName`, and `familyName` SCIM attributes and they are
    required as well. Delete the rest of the attributes.
![Okta SCIM attributes mapping](../website/assets/images/articles/okta-scim-attributes-mapping.png)


### Assign users and groups to Fleet

To send users and groups information to Fleet, you have to assign them to SCIM app that you created previously.

1. Select **Directory > Groups** in the main menu, then select **Add group**. Name it so you know that users from this group will be provisioned to Fleet (e.g "Fleet human-device mapping").
2. On the same page, select **Rules** tab. Create rule that will assign users from groups that you want to provision to Fleet to newly created "Fleet human-device mapping" group.
![Okta group rule](../website/assets/images/articles/okta-scim-group-rules.png)
3. Select **Applications > Applications** in the main menu, select app that you created previously, then select **Assignements** tab.
4. Select **Assign > Assign to Groups**, then click **Assign** next to the "Fleet human-device mapping" group, then select **Done**. Now all users that you assigned to "Fleet human-device mapping" group via rule will be provisioned to Fleet. It may take a while if you have many users.
2. On the same page, select **Push Groups** tab, then select **Push Groups > Find groups by name**,
   and add all groups that you assigned via rule to "Fleet human-device mapping" group previously (make sure that
   **Push group memberships immediately** is selected). All groups will be provisioned to
   Fleet, and Fleet will map those groups to already provisioned users .


## Connect Microsoft Entra ID

To connect Entra ID to Fleet, follow these steps:

1. Head to [Microsoft Entra admin](https://entra.microsoft.com/).
2. Select **Applications > Enterprise applications** in the main menu, select **+ New
   application**, then select **+Create your own application**.
3. Add a friendly name of the app (e.g Fleet SCIM), select **Integrate any other application you
   don't find in the gallery (Non-gallery)**, and then select **Create**.
4. In the side menu, select **Provisioning**.
5. In **Get started with application provisioning** section, select **Connect your application**.
6. In **Tenant URL**, enter `https://<your_fleet_server_url>/api/v1/fleet/scim`.
7. Create Fleet API-only user with maintainer permissions, copy API token for that user, and paste
   it in **Secret token**.
8. Select **Test connection** button. You should see success message.
9. Select **Create** and after successfull creation, you'll be on the overview page.
10. Select **Attribute mapping** from the side menu, then select **Provision Microsoft Entra ID Users**.
11. Make sure that `userName`, `givenName`, `familyName` and `active` have **Microsoft Entra ID
   Attribute** assigned to it. Currently, Fleet support `userName`, `givenName`, `familyName`, and
   `active` SCIM attributes and they are required as well. Delete the rest of the attributes.
12. Save configuration above, then select **Provision Microsoft Entra ID Groups**. Make sure that
    `displayName` and `members` have **Microsoft Entra ID Attribute** assigned to it. Delete the
    rest of attributes.
13. Then in the side menu go to **Users and groups**, select **+ Add user/group**, then select
    **None Selected** link. Select users and groups that you want to add to Fleet, then select
    **Assign**. 
14. Go to **Overview** and select **Start provisioning**.

## Verify connection in Fleet

After following steps above, you should be able to see latest requests from Okta to Fleet if you
navigate to **Settings > Integrations > Identity provider**. 

To verify that user information is added to a hosts, go to the host that has IdP email assigned, and
verify that **Full name (IdP)** and **Groups (IdP)** are populated correctly.

### Troubleshoot errors

If you find that information from IdP (e.g full name or groups) is missing on the host, and host has
IdP email assigned to it, follow steps below to resolve.

1. Please first go to Okta, select **Directory > People**, find user that is
missing information and make sure that it has all fields required by Fleet (username, first name and
last name).
2. If all required fields are present, then go to **Applications > Applications > fleet_scim_application > Provisioning > To App**, then scroll on the bottom of the page and make sure that `userName`, `givenName`, and `familyName` has value assigned to it.
3. Otherwise make sure that all settings from instructions above were set correctly.

<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2025-04-11">
<meta name="articleTitle" value="Add user information from identity provider (IdP) to host">
<meta name="category" value="guides">