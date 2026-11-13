# Windows zero touch without Entra or Autopilot

Fleet supports two ways to do [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot). The first is a [provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package): a partner preinstalls Fleet's agent (fleetd), and new computers enroll in Fleet the first time they're online. The second, starting in Fleet 5.2.0, is a Fleet factory image. It also signs the end user in with your identity provider (IdP) and installs their software before they reach the Windows desktop. Fleet builds the Windows image for you and sends it to your PC maker (OEM), and the OEM ships computers straight to your end users.

## What's new

With a [provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package), the end user creates their own Windows account during setup, and software installs after they sign in. With a Fleet factory image, all of that happens first:

| | Provisioning package | Fleet factory image |
|---|---|---|
| Who prepares the computer | Your partner applies the package | Your OEM loads Fleet's image at the factory |
| IdP sign-in | Not available | Before the first Windows sign-in |
| Setup experience software | Installs after the first sign-in | Installs before the first sign-in |
| Windows account | Created by the end user | Created by Fleet from the end user's IdP account |

## Still no Entra ID

Zero touch with Fleet doesn't use Windows Autopilot, so it doesn't need Microsoft Entra ID.

With Fleet, the computer arrives with fleetd already on it, so it knows where to enroll. Your IdP confirms who the end user is, whether that's Okta, Google Workspace, Microsoft Entra ID, or another IdP Fleet supports.

## Prerequisites

- Fleet Premium 5.2.0 (January 2027) or later, with [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup).
- Your IdP connected in **Settings > Integrations > Identity provider (IdP)**.
- Windows 11 Pro computers ordered through your OEM's custom factory image program.

## Step 1: Set up the fleet for new computers

Choose the fleet new computers should join, then set up what happens on first boot.

1. Select the fleet from the fleet dropdown.
2. Go to **Controls > Setup experience > End user authentication** and turn on end user authentication.
3. Go to **Controls > Setup experience > Install software**, select the **Windows** tab, and add the software every end user should have on day one.
4. Optional: go to **Controls > OS settings > Disk encryption** and turn on disk encryption. Fleet turns on BitLocker during setup and stores the recovery key.

Learn more in the [Windows and Linux setup experience guide](https://fleetdm.com/guides/windows-linux-setup-experience).

## Step 2: Fleet provides your image

Fleet builds a Windows 11 image with fleetd set up for that fleet, tests it, and sends it to your OEM.

> **Note:** Fleet is working on turning the factory image into a provisioning package. Today, computers get the Windows version the image was built with. With a provisioning package, they'll keep the up-to-date Windows version your OEM ships.

## Step 3: Order computers

Order computers from your OEM as usual. The OEM loads the image at the factory and ships each computer straight to the end user.

Changes you make to the fleet in Step 1, like adding software, apply to every new computer that enrolls right away. You don't need a new image.

## What end users see

1. They turn on the computer and connect to Wi-Fi.
2. Windows shows **Setting up your device** and opens your IdP's sign-in page.
3. They sign in with their work account.
4. Fleet installs their software and shows progress on screen.
5. They choose a password for their new Windows account.
6. Windows signs them out. They sign in with their new password.
7. Windows offers to set up a Windows Hello PIN.

Setup takes a few minutes, depending on how much software you add.

## Further reading

- [Autopilot without Autopilot: zero-touch Windows deployment with Fleet](https://fleetdm.com/articles/autopilot-without-autopilot)
- [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package)
- [Windows and Linux setup experience](https://fleetdm.com/guides/windows-linux-setup-experience)

<meta name="articleTitle" value="Windows zero touch without Entra or Autopilot">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-06">
<meta name="category" value="guides">
<meta name="description" value="Zero-touch Windows with Fleet can now sign end users in with your IdP and install software before their first sign-in. No Entra ID needed.">
