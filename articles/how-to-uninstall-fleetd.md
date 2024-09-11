# How to uninstall Fleet's agent (fleetd)

This guide walks you through the steps to remove fleetd from your device. After performing these steps, the device will display as an offline host in the Fleet UI until you manually remove it.

## On macOS:
Run the [cleanup script](https://github.com/fleetdm/fleet/blob/main/orbit/tools/cleanup/cleanup_macos.sh) found in Fleet's GitHub 

---

## On Windows:
Use the "Add or remove programs" dialog to remove Fleet osquery.

![windows_uninstall](https://github.com/user-attachments/assets/4140e62b-f67a-4df6-85b0-430c2c624881)

---

## On Linux:

Using Debian package manager (Debian, Ubuntu, etc.) :

Run ```sudo apt remove fleet-osquery -y```

Using yum Package Manager (RHEL, CentOS, etc.) :

Run ```sudo rpm -e fleet-osquery-X.Y.Z.x86_64```

Are you having trouble uninstalling Fleetd on macOS, Windows, or Linux? Get help on Slack in the [#fleet channel](https://fleetdm.com/slack).

<meta name="category" value="guides">
<meta name="authorFullName" value="Eric Shaw">
<meta name="authorGitHubUsername" value="eashaw">
<meta name="publishedOn" value="2021-09-08">
<meta name="articleTitle" value="How to uninstall fleetd">
<meta name="articleImageUrl" value="../website/assets/images/articles/how-to-uninstall-osquery-cover-1600x900@2x.jpg">
