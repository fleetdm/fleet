# How to uninstall osquery

This article walks you through the steps to remove osquery from your device. Remember that if you enrolled this device in a Fleet instance, it would display as an offline host in the Fleet UI until you manually remove it.

## On macOS:
Open up your terminal and paste the following commands; note that `sudo` is required, and you’ll need administrator privileges to complete this process.

```
sudo launchctl unload /Library/LaunchDaemons/io.osquery.agent.plist
sudo rm /Library/LaunchDaemons/io.osquery.agent.plist
sudo rm -rf /private/var/log/osquery /private/var/osquery
sudo rm /usr/local/bin/osquery*
sudo pkgutil --forget io.osquery.agent
```

These commands stop the running osquery daemon, remove it from your device, and delete the files created by osquery.

And that’s it; you have now removed osquery from your macOS device.

---

## On Windows:
Removing osquery on Windows 10 is a simple process. To get started, open Windows settings and go to Apps. Then find “osquery” and click Uninstall.

![Uninstall osquery](../website/assets/images/articles/how-to-uninstall-osquery-1-607x188@2x.png)

Click Uninstall again to confirm, and osquery will be removed from your Windows device. You might need to restart your computer to complete the uninstall process fully.

---

## On Linux:

1. Open your terminal and paste the following commands to stop the running osquery service, uninstall osquery, and clean up files created by osquery.

2. Note that `sudo` is required, and you’ll need administrative privileges to complete this process.

3. Using Debian package manager (Debian, Ubuntu, etc.) :

```
sudo systemctl stop osqueryd.service
sudo apt remove osquery
rm -rf /var/osquery /var/log/osquery /etc/osquery
```

Using yum Package Manager (RHEL, CentOS, etc.) :

```
sudo systemctl stop osqueryd.service
sudo yum remove osquery
rm -rf /var/osquery /var/log/osquery /etc/osquery
```

Are you running into trouble uninstalling osquery on macOS, Windows, or Linux? Get help on Slack in the [#fleet channel](https://fleetdm.com/slack).

<meta name="category" value="guides">
<meta name="authorFullName" value="Eric Shaw">
<meta name="authorGitHubUsername" value="eashaw">
<meta name="publishedOn" value="2021-09-08">
<meta name="articleTitle" value="How to uninstall osquery">
<meta name="articleImageUrl" value="../website/assets/images/articles/how-to-uninstall-osquery-cover-1600x900@2x.jpg">
