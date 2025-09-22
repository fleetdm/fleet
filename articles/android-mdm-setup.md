# Android MDM setup

> Experimental feature. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

This guide provides instructions to turn on Android MDM features by connecting Fleet to Android Enterprise.

Fleet supports Android devices that are [Play Protect certified](https://support.google.com/googleplay/answer/7165974?hl=en) (previously known as GMS).

## Turn on

To turn on Android MDM, connect Android Enterprise on **Settings > Integrations > Mobile device management (MDM)** page.

When you select **Connect Android Enterprise**, Fleet will open the Google signup page. The signup process varies depending on whether your organization uses [Google Workspace](#google-workspace), [Microsoft 365](#microsoft-365), or [another provider](#other). Organizations using Google Workspace and Microsoft don't need to verify domain ownership.

### Google Workspace

1. If your organization already uses Google Workspace, use your admin account to signup for Android Enterprise. If you don't know your admin account credentials, ask your Google Workspace admin.
2. Follow the steps in Google's signup flow.
3. After successful signup, a free Android Enterprise subscription is added to your Google Workspace. In Fleet, you can confirm Android MDM is turned on in **Settings > Integrations > MDM**.

### Microsoft 365

1. If your organization uses Microsoft 365, you can use your Microsoft email to signup for Android Enterprise. After you select **Connect Android Enterprise**, select **Sign in with Microsoft**. Your Microsoft account must have access to an email.
2. Follow the steps in Google's signup flow.
3. After successful signup, a free Android Enterprise subscription is added to your Google Workspace. In Fleet, you can confirm Android MDM is turned on in **Settings > Integrations > MDM**.
4. Go to your [Google Admin console](https://admin.google.com).
5. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name. This way, only you can use your domain to sign up for Google Workspace.

Now you have managed Google domain with an Android Enterprise subscription. Optionally, if you want to add additional subscriptions later (i.e. Google Workspace) you can use this domain. Only the free Android Enterprise subscription is required for Android MDM features.

### Other

1. If your organization doesn't use Google Workspace or Microsoft 365, in the Google signup page, use a work email to signup for Android Enterprise (don't use personal emails like "@gmail.com").
2. After you enter your email, you'll get a verification email. Open the link from the email.
3. Enter information about you and your company and select **Continue**.
4. You'll see that your free Android Enterprise subscription will be selected. Select **Next**.
5. Enter a password for your account and select **Agree and continue**.
6. Select **Allow and create account** on the next screen.
8. You'll be asked to log in with your account that you just created and confirm your phone number.
9. After successful login and phone verification, you'll be redirected to Fleet. In Fleet, you can confirm Android MDM is turned on in **Settings > Integrations > MDM**.
10. Follow [these steps](https://support.google.com/a/answer/60216?hl=en) to verify your domain name. This way, only you can use your domain to sign up for Google Workspace.

Now you have managed Google domain with an Android Enterprise subscription. Optionally, if you want to add additional subscriptions later (i.e. Google Workspace) you can use this domain. Only the free Android Enterprise subscription is required for Android MDM features.

## Enrollment

Learn how to enroll Android hosts in the [enroll hosts guide](https://fleetdm.com/guides/enroll-hosts#ui).

## Migration

To migrate hosts from other MDM solution, you must first unenroll hosts from your old solution and share a link with your end users so they can enroll to Fleet. Learn how to find your enrollment link in the [enroll hosts guide](https://fleetdm.com/guides/enroll-hosts#ui).

## Turn off

1. In Fleet, head to **Settings > Integrations > MDM**.
2. In the **Mobile Device Management (MDM)** section, select **Edit** next to "Android MDM turned on."
3. Select **Turn off Android MDM**

When you turn off Android MDM in Fleet, your Android Enterprise will be deleted, MDM will be turned off on all hosts, and the work profile will be deleted from all Android hosts.

If you ever delete your Android Enterprise in your [Google Admin console](https://admin.google.com) instead of in Fleet, Android MDM will be turned off in Fleet, and the work profile will be deleted from all Android hosts.

<meta name="articleTitle" value="Android MDM setup">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-04-05">
<meta name="description" value="Learn how to turn on Android MDM in Fleet.">
