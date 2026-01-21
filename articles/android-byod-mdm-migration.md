# Android BYOD MDM migration

On BYOD Android devices, enrolling in an MDM installs a [Work Profile](https://support.google.com/work/android/answer/6191949?hl=en), which segments corporate apps and data from the end user's personal information.

Before you can enroll Android devices in Fleet, you must [set up Android MDM](https://fleetdm.com/guides/android-mdm-setup).

Add Android hosts to Fleet using an enrollment link. Follow our [Enroll hosts guide](https://fleetdm.com/guides/enroll-hosts#ui) for instructions on how to get this link.


## Remove the old Work Profile

To migrate from another MDM to Fleet, you must first remove the exisiting Work Profile.

On Google Pixel devices, open **Settings** > **Passwords, passkeys & accounts** > **Work** > **Remove work profile**.

<img width="2500" height="2000" alt="image5" src="https://github.com/user-attachments/assets/5119dffb-c00d-45c0-a0a6-d53d79bd2ce5" />

<img width="50%" alt="50" src="https://github.com/user-attachments/assets/f6580469-89d2-423a-9617-8626fe44a9c3" />

On Samsung devices, open **Settings** > **Accounts and backup** > **Manage accounts** > **Work** > **Uninstall Work profile**.

<img width="2500" height="2000" alt="image0" src="https://github.com/user-attachments/assets/074e1a65-dbcf-4f72-99f9-5cec7125db1b" />

<img width="2500" height="2000" alt="image1" src="https://github.com/user-attachments/assets/3a711cf4-37e7-437c-b7d0-35f2b9c91b48" />

After selecting **Delete** on the confirmation dialog, the old Work Profile will be removed from the device.


## Enroll in Fleet

Open the enrollment link in a web browser. If [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication) is set up for the team, end users will be prompted to authenticate through SSO. We use Google for our setup, so this step may look different in your environment.

<img width="50%" alt="Screenshot_20260120-195230" src="https://github.com/user-attachments/assets/b9a92a3c-c7d6-4ff9-8c9b-3fd478c53460" />

After successfully authenticating, a page with an Enroll button will appear. Select **Enroll**.

<img width="50%" alt="51" src="https://github.com/user-attachments/assets/3ca1c1dd-79dd-4ca6-ab4b-a0e32da834a1" />

A Set up your work profile screen will then appear. Select **Next**, then the next screen will describe what a Work Profile is: select **Accept & continue** here (on Samsung devices, this is **Agree**).

<img width="2500" height="2000" alt="image6" src="https://github.com/user-attachments/assets/5f09407a-4db3-4ed3-ac60-ab11c6c7cb1e" />

The Work Profile setup will then begin, and on Samsung devices there may be one more prompt to select **Next** at.

<img width="2500" height="2000" alt="image3" src="https://github.com/user-attachments/assets/e6823e4a-ed66-4831-8c63-f5b663e7a0a3" />

A series of enrollment screens will appear. A briefcase icon appears in the status bar on Google Pixel devices, and in the lower right corner on Samsung devices, when the Work Profile is active.

<img width="2500" height="2000" alt="image7" src="https://github.com/user-attachments/assets/3b7e51f9-ca35-459d-8ca7-77a9c3b767bf" />

If Google authentication is enabled in [Google Workspace](https://support.google.com/work/android/answer/9415508?hl=en), the end user will be prompted to sign in to their work Google account. If a user selects Skip at this screen, they will later be required to sign in to this Google account to access apps like Google Calendar.

<img width="50%" alt="27" src="https://github.com/user-attachments/assets/491d381b-b224-4d7e-8ae3-99f82b5c3661" />

When enrollment is complete, the Work Profile screens will go away and the end user will be brought back to the web browser with the Fleet enrollment page.

Open the App Drawer (swipe up at the home screen, or select the Apps icon), and there will now be a separate tab at the bottom for Work Profile apps. Work profile apps have a briefcase icon in the bottom right corner of their icon.

<img width="50%" alt="30" src="https://github.com/user-attachments/assets/93491232-8063-42b5-9971-fb190071e439" />

If the device doesn't meet the requirements set up by the admin in Google Workspace, the end user will be prompted to resolve these.

<img width="2500" height="2000" alt="image8" src="https://github.com/user-attachments/assets/294c96e3-db11-4813-8864-0eaf428eba8d" />


<meta name="articleTitle" value="Android BYOD MDM migration">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-20">
<meta name="description" value="Instructions for migrating Android BYOD hosts away from an old MDM solution to Fleet.">
