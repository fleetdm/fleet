# Android MDM setup

To turn on Android MDM features, follow the instructions on this page to connect Android Enterprise
to Fleet.

Fleet supports Android devices that are [Play Protect certified](https://support.google.com/googleplay/answer/7165974?hl=en) (previously known as GMS).

## Turn Android MDM

To turn on Android MDM, connect Android Enterprise on **Settings > Integrations > Mobile device management (MDM)** page.

When you select **Connect Android Enterprise**, Fleet will open the Google sign-up page. The registration process varies depending on whether your organization uses [Google Workspace](#google-workspace), [Microsoft 365](#microsoft-365), or [another provider](#other). Organizations using Google Workspace and Microsoft don't need to verify domain ownership.

### Google Workspace

1. If your organization already uses Google Workspace, use the admin account (or if you don't have one ask your Google Workspace admin to create one for you) to register Android Enterprise.
2. Follow the steps in Google's sign-up flow to finish the registration.
3. After successful registration Android Enterprise subscription (free) is added to your Google Workspace, and you can see in Fleet that Android MDM is turned on.
4. Go to [Google Admin console](https://admin.google.com)
5. From the side menu, select **Devices > Mobile & endpoints > Settings > Third-party integrations**.
6. Select **Android EMM**, check **Enable third-party Android mobile management**, then select **Manage EMM providers**.
7. Toggle **Authenticate Using Google** switch for your Android Enterprise, select the cross icon in the top left corner, and select **Save**.

### Microsoft 365

1. If your organization uses Microsoft 365, you can use your email to register Android Enterprise. When you click **Connect Android Enterprise**, you'll see option to **Sign in with Microsoft**.
2. Follow the steps in Google's sign-up flow to finish the registration.
3. After successful registration Android Enterprise subscription (free) is added to your Google Workspace, and you can see in Fleet that Android MDM is turned on.

### Other

1. If your organization doesn't use Google Workspace or Microsoft 365, in the Google sign-up page, use a work email to register Android Enterprise (don't use personal emails like "@gmail.com").
2. After you enter your email, you'll get a verification email. Open the link from the email.
3. Enter information about you and your company and select **Continue**.
4. Android Enterprise subscription will be selected (free), select **Next**.
5. Enter a password for your account and select **Agree and continue**.
6. Select **Allow and create account** on the next screen.
8. You'll be asked to log in with your account that you just created, and to confirm your phone number.
9. After successful login and phone verification, you'll be redirected to Fleet, and you should see that Android MDM is turned on.
10. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name and prevent others from signing up with your domain.

This way, you created Managed Google Domain, with Android Enterprise subscription only. You can use this domain later to add additional subscriptions (i.e. Google Workspace) if you need.

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
