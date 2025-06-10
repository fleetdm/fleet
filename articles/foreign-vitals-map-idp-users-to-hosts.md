# Foreign vitals: map IdP users to hosts

![Import users from IdP to Fleet](../website/assets/images/articles/add-users-from-idp-cover-img.png)

_Available in Fleet Premium._

To add IdP host vitals, like the end user's groups and full name, follow steps for your IdP. 

Fleet currently supports [Okta](#okta). [Microsoft Active Directory (AD) / Entra ID](#microsoft-entra-id), [Google Workspace](#google-workspace), and [authentik](#google-workspace), more are coming soon.


## Okta

To map users from Okta to hosts in Fleet, do the following steps:

- [Create application in Okta](#step-1-create-application-in-okta)
- [Connect Okta to Fleet](#step-2-connect-okta-to-fleet)
- [Map users and groups to hosts in Fleet](#step-3-map-users-and-groups-to-hosts-in-fleet)

#### Step 1: Create application in Okta

1. Head to Okta admin dashboard.
2. In the main menu, select **Applications > Applications**, then select **Create App Integration**.
3. Select **SAML 2.0** option and select **Next**.
4. On the **General Settings** page, add a friendly **App name** (e.g Fleet SCIM), and select **Next**.
5. On the **SAML Settings** page, add any URL to the **Single sign-on URL** and **Audience URI (SP Entity ID)** fields, and select **Next**.
> Okta requires us to setup SAML settings in order to setup a SCIM integration. Since we don't need SAML right now, you can set the URL to anything like "example.fleetdm.com".
6. On the **Feedback** page, provide feedback if you want, and select **Finish**.
7. Select the **General** tab in your newly created app and then select **Edit** in **App Settings**.
8. For **Provisioning**, select **SCIM** and select **Save**.

#### Step 2: Connect Okta to Fleet

1. Select the **Provisioning** tab and then, in **SCIM Connection**, select **Edit**.
2. For the **SCIM connector base URL**, enter `https://<your_fleet_server_url>/api/v1/fleet/scim`.
3. For the **Unique identifier field for users**, enter `userName`.
4. For the **Supported provisioning actions**, select **Push New Users**, **Push Profile Updates**, and **Push Groups**.
5. For the **Authentication Mode**, select **HTTP Header**.
6. [Create a Fleet API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with maintainer permissions and copy API token for that user. Paste your API token in Okta's **Authorization** field.
7. Select the **Test Connector Configuration** button. You should see success message in Okta.
8. In Fleet, head to **Settings > Integrations > Identity provider (IdP)** and verify that Fleet successfully received the request from IdP.
9. Back in Okta, select **Save**.
10. Under the **Provisioning** tab, select **To App** and then select **Edit** in the **Provisioning to App** section. Enable **Create Users**, **Update User Attributes**, **Deactivate Users**, and then select **Save**.
11. On the same page, make sure that `givenName` and `familyName` have Okta value assigned to it. Currently, Fleet requires the `userName`, `givenName`, and `familyName` SCIM attributes. Delete the rest of the attributes.
![Okta SCIM attributes mapping](../website/assets/images/articles/okta-scim-attributes-mapping.png)


#### Step 3: Map users and groups to hosts in Fleet

To send users and groups information to Fleet, you have to assign them to your new SCIM app.

1. In OKta's main menu **Directory > Groups** and then select **Add group**. Name it "Fleet human-device mapping".
2. On the same page, select the **Rules** tab. Create a rule that will assign users to your  "Fleet human-device mapping" group.
![Okta group rule](../website/assets/images/articles/okta-scim-group-rules.png)
3. In the main menu, select **Applications > Applications**  and select your new SCIM app. Then, select the **Assignments** tab.
4. Select **Assign > Assign to Groups** and then select **Assign** next to the "Fleet human-device mapping" group. Then, select **Done**. Now all users that you assigned to the  "Fleet human-device mapping" group will be provisioned to Fleet.
5. On the same page, select **Push Groups** tab. Then, select **Push Groups > Find groups by name** and add all groups that you assigned to "Fleet human-device mapping" group previously (make sure that **Push group memberships immediately** is selected). All groups will be provisioned in Fleet, and Fleet will map those groups to users.

## Verify connection

After following steps above, you should be able to see latest requests from your IdP to Fleet if you navigate to **Settings > Integrations > Identity Provider (IdP)**. 

To verify that user information is added to a host, go to the host that has IdP username assigned, and verify that **Full name (IdP)** and **Groups (IdP)** are populated correctly.

> Currently, the IdP username is only supported on macOS hosts. It's collected once, during automatic enrollment (DEP), only if the [end user authenticates](https://fleetdm.com/docs/rest-api/rest-api#mdm-macos-setup) with the IdP and the DEP profile has `await_device_configured` set to `true` (default in the [automatic enrollment profile](https://fleetdm.com/guides/macos-setup-experience#step-1-create-an-automatic-enrollment-profile)).

### Troubleshooting

If you find that information from IdP (e.g full name or groups) is missing on the host, and host has IdP username assigned to it, follow steps below to resolve.

1. Please first go to Okta, select **Directory > People**, find user that is
missing information and make sure that it has all fields required by Fleet (username, first name, and
last name).
2. If all required fields are present, then go to **Applications > Applications > fleet_scim_application > Provisioning > To App**, then scroll on the bottom of the page and make sure that `userName`, `givenName`, and `familyName` has value assigned to it.
3. Otherwise make sure that all settings from instructions above were set correctly.

<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2025-04-11">
<meta name="articleTitle" value="Foreign vitals: map IdP users to hosts">
<meta name="articleImageUrl" value="../website/assets/images/articles/add-users-from-idp-cover-img.png">
<meta name="category" value="guides">
