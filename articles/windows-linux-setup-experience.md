# Windows & Linux setup experience

_Available in Fleet Premium_

In Fleet, you can customize the out-of-the-box Windows and Linux setup.

Here's what you can configure, and in what order each happen, to your Windows and Linux hosts during setup:

1. Require [end users to authenticate](#end-user-authentication) with your identity provider (IdP).

2. [Install software](#install-software) (App Store apps, custom packages, and Fleet-maintained apps). 

## End user authentication

### End user experience

Fleet automatically opens the default web browser and directs the end user to log in before the setup process can continue.

TODO screenshots

> If the Fleet agent (fleetd) installed is older than version 1.50.0, end user authentication won't be enforced.

### Setup

TODO steps to configure

## Install software

### End user experience

Fleet automatically opens the default web browser to show end users software install progress:

![screen shot of Fleet setup experience webpage](../website/assets/images/articles/setup-experience-browser-1795x1122@2x.png)

The browser can be closed, and the installation will continue in the background. End users can return to the setup experience page by clicking **My Device** from Fleet Desktop.  Once all steps have completed, the **My Device** page will show the host information as usual.

If software installs fail, Fleet automatically retries. Learn more in the [macOS setup experience guide](https://fleetdm.com/guides/macos-setup-experience#install-software).

To replace the Fleet logo with your organization's logo:

1. Go to **Settings** > **Organization settings** > **Organization info**
2. Add URLs to your logos in the **Organization avatar URL (for dark backgrounds)** and **Organization avatar URL (for light backgrounds)** fields
3. Press **Save**

> See [configuration documentation](https://fleetdm.com/docs/configuration/yaml-files#org-info) for recommended logo sizes.

> Software installations during setup experience are automatically attempted up to 3 times (1 initial attempt + 2 retries) to handle intermittent network issues or temporary failures. This ensures a more reliable setup process for end users.

### Add software

Add setup experience software setup experience:

1. Click on the **Controls** tab in the main navigation bar,  then **Setup experience** > **3. Install software**.
2. Click on the tab corresponding to the operating system (e.g. Linux).
3. Click **Add software**, then select or search for the software you want installed during the setup experience.
4. Press **Save** to save your selection.

Fleet also provides a API endpoints for managing setup experience software programmatically. Learn more in Fleet's [API reference](https://fleetdm.com/docs/rest-api/rest-api#update-software-setup-experience).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="dantecatalfamo">
<meta name="authorFullName" value="Dante Catalfamo">
<meta name="publishedOn" value="2025-09-24">
<meta name="articleTitle" value="Windows & Linux setup experience">
<meta name="description" value="Install software when Linux and Windows workstations enroll to Fleet">
