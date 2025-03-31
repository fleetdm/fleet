# Android MDM setup

To turn on Android MDM features, follow the instructions on this page to connect Android Enterprise to Fleet.

## Turn Android MDM

To turn on Android MDM, connect Android Enterprise, on **Settings > Integrations > Mobile device management (MDM)** page.

When you select **Connect Android Enterprise**, Fleet will open Google sign-up page. Depending whether your organization already use Google Workspace, registration process is different.

### Organizations using Google Workspace

1. If your organization already uses Google Workspace, use admin account (or if you don't have one ask your Google Workspace admin to create one for you) to register Android Enterprise.
2. Follow steps in Google's sign-up flow to finish registration.
3. After successfull registration Android Enterprise subscription (free) is added to your Google Workspace, and you can see in Fleet that Android MDM is turned on.
4. Go to [Google Admin console](https://admin.google.com)
5. From the side menu, select **Devices > Mobile & endpoints > Settings > Third-party integrations**.
6. Select **Android EMM**, check **Enable third-party Android mobile management**, then select **Manage EMM providers**.
7. Toggle **Authenticate Using Google** switch for your Android Enterprise, select cross icon in the top left corner, and select **Save**.

### Organizations that don't use Google Workspace

1. In Google sign-up page use work email to register Android Enterprise (don't use personal emails like "@gmail.com").
2. After you entered your email, you'll get verification email. Open link from the email.
3. Enter information about you and company and select **Continue**.
4. Android Enterprise subscription will be selected (free), select **Next**.
5. Enter password for your account, and select **Agree and continue**.
6. Select **Allow and create account** on the next screen.
8. You'll be asked to log in with your account that you just created, and to confirm your phone number.
9. After succussfull login and phone verification, you'll be redirected to Fleet, and you should see that Android MDM is turned on.
10. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name, and prevent others from signing up with your domain.

This way you created Managed Google Domain, with Android Enterprise subscription only. You can use this domain later to add additional subscriptions (i.e. Google Workspace) if you need.

## Turn off Android MDM

1. Head to **Settings > Integrations > MDM**.
2. In the **Mobile Device Management (MDM)** section, select **Edit** next to "Android MDM turned on."
3. Select **Turn off Android MDM**

When you turn off Android MDM, your Android Enterprise will be deleted, and MDM will be turned off
on all hosts.


<meta name="articleTitle" value="Android MDM setup">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-04-05">
<meta name="description" value="Learn how to turn on Android MDM in Fleet.">
