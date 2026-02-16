# Android BYOD MDM migration

On BYOD Android devices, enrolling in an MDM installs a [Work Profile](https://support.google.com/work/android/answer/6191949?hl=en), which segments corporate apps and data from the end user's personal information.

Before you can enroll Android devices in Fleet, you must [set up Android MDM](https://fleetdm.com/guides/android-mdm-setup).

Add Android hosts to Fleet using an enrollment link. Follow our [Enroll hosts guide](https://fleetdm.com/guides/enroll-hosts#ui) for instructions on how to get this link.


## Remove the old Work Profile

To migrate from another MDM to Fleet, you must first remove the existing Work Profile.

1. On Google Pixel devices, open **Settings** > **Passwords, passkeys & accounts** > **Work** > **Remove work profile**.
   - On Samsung devices, open **Settings** > **Accounts and backup** > **Manage accounts** > **Work** > **Uninstall Work profile**.
2. After selecting **Delete** on the confirmation dialog, the old Work Profile will be removed from the device.


## Enroll in Fleet

Send the enrollment link to end users to open in a web browser. An easy alternative is to use a QR code. To generate one using Chrome, open the enrollment link on a computer, right-click the page, then select **Create QR Code for this Page**. If this isn't showing, select the three dot menu icon on the right side of the toolbar > **Cast, Save, and Share** > **Create QR Code**.

1. Open the enrollment link on the Android device.
   - If [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication) is set up for the team, end users will be prompted to authenticate through SSO. After successfully authenticating, a page with an Enroll button will appear.
2. Select **Enroll**.
3. A "Set up your work profile" screen will then appear. Select **Next**, then the next screen will describe what a Work Profile is: select **Accept & continue** here (on Samsung devices, this is **Agree**).
   - The Work Profile setup will then begin, and on Samsung devices, there may be one more prompt to select **Next**.
4. A series of enrollment screens will appear. A briefcase icon appears in the status bar on Google Pixel devices, and in the lower right corner on Samsung devices, when the Work Profile is active.
5. If Google authentication is enabled in [Google Admin](https://support.google.com/work/android/answer/9415508?hl=en), the end user will be prompted to sign in to their work Google account. If a user selects Skip at this screen, they will later be required to sign in to this Google account to access apps like Google Calendar.
6. When enrollment is complete, the Work Profile screens will go away and the end user will be brought back to the web browser with the Fleet enrollment page.

Open the App Drawer (swipe up at the home screen, or select the Apps icon), and there will now be a separate tab at the bottom for Work Profile apps. Work profile apps have a briefcase icon in the bottom right corner of their icon.

When an end user signs in to their Google account, if the device doesn't meet the requirements set up by the admin in Google Admin, the end user will be prompted to resolve these.


<meta name="articleTitle" value="Android BYOD MDM migration">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-20">
<meta name="description" value="Instructions for migrating Android BYOD hosts away from an old MDM solution to Fleet.">
