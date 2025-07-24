# Enroll personal (BYOD) iPhones and iPads with Managed Apple Account

_Available in Fleet Premium._

To enroll personal IPhones and iPads with [Account-driven User Enrollment](https://support.apple.com/en-gb/guide/deployment/dep23db2037d/web), and provide seamless experience for your end users, follow steps below.
This approach allows both a Managed Apple Account and a personal Apple Account to be signed in on the same device, with complete separation of work and personal data. Users maintain privacy over their personal information, and IT manages work-related OS settings and apps.

- Step 1: Add Apple Business Manager (ABM) to Fleet
- Step 2: Add and verify your domain in Apple Business Manager (ABM)
- Step 3: Federate your IdP accounts to Apple Business Manager (ABM)
- Step 4: Host service discovery file (optional)
- Step 5: Login to enroll to Fleet (end user experience)


## Step 1: Add Apple Business Manager (ABM) to Fleet

1. Follow instructions to add ABM to Fleet.
2. In ABM, select **Your name > Preferences > Management Assignment**. Then, choose **Edit** and set the Fleet MDM server, created in the first step, as the default for iPhone and iPad.

> Fleet MDM server must be default for iPhone and iPad for User Enrollment to work.

## Step 2: Add and verify your domain in Apple Business Manager (ABM)

Follow [Apple documentation](https://support.apple.com/en-gb/guide/apple-business-manager/axm48c3280c0/web#axm2033c47b0) to add and verify your company domain to your ABM. Add domain name that's used in your work email (yourcompany.com from name@yourcompany.com). This will enable automatic creation of Apple Managed Accounts from your identitity provider (IdP) accounts in the next step.

## Step 3: Federate your IdP accounts to Apple Business Manager (ABM)

Follow [Apple documentation](https://support.apple.com/en-gb/guide/apple-business-manager/axmb19317543/web) to connect your identity provider (IdP) to enable end users to login to Managed Apple Account with existing IdP credentials.

You can watch these videos as well:
 - [Connect Google Workspace to ABM](https://www.youtube.com/watch?v=CPfO6W67d3A)
 - [Connect Microsoft Entra ID to ABM](https://www.youtube.com/watch?v=_-PnhMurAVk)

## Step 4: Host service discovery file (optional)

Fleet manages service discovery for hosts that run iOS 18.2/iPadOS 18.2 and above. For hosts below these versions, you need to self-host service discovery file on your company domain that you added to ABM in the second step above.

// TODO: JSON that needs to be hosted and instructions that server must return the `Content-Type: application/json` header with the file

## Step 5: Login to enroll to Fleet (end user experience)

// TODO: Steps for the end user. Go to Settings > General > ... > Login with company email > ...

// TODO: update https://fleetdm.com/guides/macos-mdm-setup#automatic-enrollment