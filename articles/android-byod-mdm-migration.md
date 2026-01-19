# Android BYOD MDM migration

On BYOD Android devices, enrolling in an MDM installs a _Work Profile_, which segments corporate apps and data from the end user's personal information.

Before you can enroll Android devices in Fleet, you must [set up Android MDM](https://fleetdm.com/guides/android-mdm-setup).

Enroll Android devices into Fleet using an enrollment link. Follow our [Enroll hosts guide](https://fleetdm.com/guides/enroll-hosts#ui) for instructions on how to get this link.


## Remove the old Work Profile

To migrate from another MDM to Fleet, you must first remove the exisiting Work Profile.

Open Settings, scroll down and select Accounts and backup, then Manage accounts.

<img width="2500" height="2000" alt="image0" src="https://github.com/user-attachments/assets/074e1a65-dbcf-4f72-99f9-5cec7125db1b" />

At the bottom of the screen, you'll see two tabs: Personal and Work. Select Work, then select Uninstall Work profile.

<img width="2500" height="2000" alt="image1" src="https://github.com/user-attachments/assets/3a711cf4-37e7-437c-b7d0-35f2b9c91b48" />

After selecting Delete on the confirmation dialog, the old Work Profile will be removed.


## Enroll in Fleet

Open the enrollment link in a web browser. If [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication) is set up for the team, end users will be prompted to authenticate through SSO. We use Google for our setup, so this step may look different in your environment.

<img width="50%" alt="1000000057" src="https://github.com/user-attachments/assets/ea37c648-9f47-4813-bf73-e17881943488" />

After successfully authenticating, a page with an Enroll button will appear. Select Enroll.

<img width="50%" alt="1000000060" src="https://github.com/user-attachments/assets/4720302f-c7a0-4237-a4c8-9a8ebedea50e" />

A Set up your work profile screen will then appear. Select Next, then the next screen will describe what a Work Profile is: select Agree here.

<img width="2500" height="2000" alt="image2" src="https://github.com/user-attachments/assets/29587f35-94b9-4775-ad98-1f464bedee69" />

The Work Profile setup will then begin, and there will be one more prompt to select Next at.

<img width="2500" height="2000" alt="image3" src="https://github.com/user-attachments/assets/e6823e4a-ed66-4831-8c63-f5b663e7a0a3" />

A series of enrollment screens will appear. A briefcase icon appears in the lower right corner when the Work Profile is active.

<img width="2500" height="2000" alt="image4" src="https://github.com/user-attachments/assets/6090ca86-668e-4033-9a69-3bdd6d8f664c" />

**TODO: If Google authentication is enabled (TODO: INSERT LINK), the end user will be prompted to sign in to their work Google account.**

**TODO: SCREENSHOT**

**TODO: If the device doesn't meet the requirements set up by the admin, the end user will be prompted to resolve these.**

**TODO: SCREENSHOT**

When enrollment is complete, the Work Profile screens will go away and the end user will be brought back to the web browser with the Fleet enrollment page.

Open the App Drawer on the home screen (either with the Apps icon, or by swiping up at the home screen), and there will now be a separate tab at the bottom for Work Profile apps. Work profile apps have a briefcase icon in the bottom right corner of their icon.

<img width="50%" alt="10000000691" src="https://github.com/user-attachments/assets/ebc29262-5edb-4e9d-9dbe-cbc17e80213a" />
