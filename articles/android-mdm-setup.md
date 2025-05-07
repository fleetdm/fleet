# Android MDM setup

> Experimental feature. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

Android MDM features are currently behind a feature flag. To enable them, set `ANDROID_FEATURE_ENABLED=1` in your [server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration).

This guide provides instructions to turn on Android MDM features by connecting Fleet to Android Enterprise.
to Fleet.

Fleet supports Android devices that are [Play Protect certified](https://support.google.com/googleplay/answer/7165974?hl=en) (previously known as GMS).

## Turn Android MDM

To turn on Android MDM, connect Android Enterprise on **Settings > Integrations > Mobile device management (MDM)** page.

When you select **Connect Android Enterprise**, Fleet will open the Google signup page. The signup process varies depending on whether your organization uses [Google Workspace](#google-workspace), [Microsoft 365](#microsoft-365), or [another provider](#other). Organizations using Google Workspace and Microsoft don't need to verify domain ownership.

### Google Workspace

1. If your organization already uses Google Workspace, use the admin account (or if you don't have one ask your Google Workspace admin to create one for you) to signup for Android Enterprise.
2. Follow the steps in Google's signup flow.
3. After successful signup, Android Enterprise subscription (free) is added to your Google Workspace, and you can see in Fleet that Android MDM is turned on.
4. Go to [Google Admin console](https://admin.google.com)
5. From the side menu, select **Devices > Mobile & endpoints > Settings > Third-party integrations**.
6. Select **Android EMM**, check **Enable third-party Android mobile management**, then select **Manage EMM providers**.
7. Toggle **Authenticate Using Google** switch for your Android Enterprise, select the cross icon in the top left corner, and select **Save**.

### Microsoft 365

1. If your organization uses Microsoft 365, you can use your Microsoft email to signup for Android Enterprise. When you select **Connect Android Enterprise**, you'll see option to **Sign in with Microsoft**. Your Microsoft account must have access to an email.
2. Follow the steps in Google's signup flow.
3. After successful signup, Android Enterprise subscription (free) is added to your Google Workspace, and you can see in Fleet that Android MDM is turned on.
4. Go to [Google Admin console](https://admin.google.com)
5. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name and prevent others from signing up with your domain.

Now you have managed Google domain with an Android Enterprise subscription. Optionally, if you want to add additional subscriptions later (i.e. Google Workspace) you can use this domain. Only the Android Enterprise subscription is required for Android MDM features.

#### Add users from Microsoft to Google Workspace

To enable end user authentication with Google account and sync your Microsoft accounts to Google Workspace, follow steps below:

1. From the side menu, select **Devices > Mobile & endpoints > Settings > Third-party integrations**.
2. Select **Android EMM**, check **Enable third-party Android mobile management**, then select **Manage EMM providers**.
3. Toggle **Authenticate Using Google** switch for your Android Enterprise, select the cross icon in the top left corner, and select **Save**.
4. From the side menu, select **Directory > Directory Sync**, and select **Add Azure Active Directory** to sync users from your Microsoft 365 to Google Workspace, so your end user can login with Microsoft 365 to add work profile.
5. Select **Continue**, add name and description, then select **Authorize and save**.
6. In popup window, login with Microsoft 365 admin account, select **Consent on behalf of your organisation**, and select **Accept**.
7. When you see **Connection successful** page, select **Continue**, and on the directory sync details page, select **Set up user sync**.
8. Enter names of the groups that you want to sync from Microsoft 365, then select **Verify**, and after that, select **Continue**.
9. Now choose organizational unit to add users to by selecting **Select organizational unit** button, then select **Continue**
10. You can keep default user attribute mapping and select **Continue**, select **Don't send activation email** and select **Continue**.
11. Keep **Suspend user in Google Directory** checked and select **Continue**
12. You can keep default safeguards and select **Simulate sync**, then after successfull simulation, select **Close**. This can [take up to the hour](https://support.google.com/a/answer/10344342) to complete, depending on the size of your data.
13. In the dialog, select **Activate and start sync**.

### Other

1. If your organization doesn't use Google Workspace or Microsoft 365, in the Google signup page, use a work email to signup for Android Enterprise (don't use personal emails like "@gmail.com").
2. After you enter your email, you'll get a verification email. Open the link from the email.
3. Enter information about you and your company and select **Continue**.
4. Android Enterprise subscription will be selected (free), select **Next**.
5. Enter a password for your account and select **Agree and continue**.
6. Select **Allow and create account** on the next screen.
8. You'll be asked to log in with your account that you just created, and to confirm your phone number.
9. After successful login and phone verification, you'll be redirected to Fleet, and you should see that Android MDM is turned on.
10. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name and prevent others from signing up with your domain.

Now you have managed Google domain with an Android Enterprise subscription. Optionally, if you want to add additional subscriptions later (i.e. Google Workspace) you can use this domain. Only the Android Enterprise subscription is required for Android MDM features.

## Turn off Android MDM

1. In Fleet, head to **Settings > Integrations > MDM**.
2. In the **Mobile Device Management (MDM)** section, select **Edit** next to "Android MDM turned on."
3. Select **Turn off Android MDM**

When you turn off Android MDM, your Android Enterprise will be deleted, and MDM will be turned off
on all hosts. The work profile from all BYOD hosts is deleted.


<meta name="articleTitle" value="Android MDM setup">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-04-05">
<meta name="description" value="Learn how to turn on Android MDM in Fleet.">
