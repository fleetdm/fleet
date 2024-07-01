# Config-less `fleetd` agent deployment

![Config-less `fleetd` agent deployment](../website/assets/images/articles/config-less-fleetd-agent-deployment-1600x900@2x.png)

Deploying Fleet's agent across a diverse range of devices often involves the crucial step of enrolling each device. Traditionally, this involves [packaging](https://fleetdm.com/docs/using-fleet/fleetd#packaging)  `fleetd` with configuration including the enroll secret and server URL. While effective, an alternative offers more flexibility in your deployment process. This guide introduces a different approach for deploying Fleet's agent without embedding configuration settings directly into `fleetd`. Ideal for IT administrators who prefer to generate a single package and maintain greater control over the distribution of enrollment secrets and server URLs, this method simplifies the enrollment process across macOS and Windows hosts.

Emphasizing adaptability and convenience, this approach allows for a more efficient way to manage device enrollments. Let’s dive into how to deploy Fleet's agent using this alternative method, ensuring a more open and flexible deployment process.


## For macOS:

1. First, you need to build an installer that will read the configs from an enrollment profile using:


```
fleetctl package --type=pkg --use-system-configuration --fleet-desktop
```


> [Download the latest version of fleetctl.](https://github.com/fleetdm/fleet/releases/latest)

2. With your MDM, send an enrollment configuration profile like the example provided here (be sure to replace `YOUR_ENROLL_SECRET_HERE` and `YOUR_FLEET_URL_HERE` with proper values.):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <array>
      <dict>
        <key>EnrollSecret</key>
        <string>YOUR_ENROLL_SECRET_HERE</string>
        <key>FleetURL</key>
        <string>YOUR_FLEET_URL_HERE</string>
        <key>PayloadDisplayName</key>
        <string>Fleetd configuration</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.fleetd.config</string>
        <key>PayloadType</key>
        <string>com.fleetdm.fleetd.config</string>
        <key>PayloadUUID</key>
        <string>476F5334-D501-4768-9A31-1A18A4E1E807</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
      </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Fleetd configuration</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.fleetd.config</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadDescription</key>
    <string>Default configuration for the fleetd agent.</string>
  </dict>
</plist>
```


## For Windows:

1. Download the Base MSI installer from [https://download.fleetdm.com/stable/fleetd-base.msi](https://download.fleetdm.com/stable/fleetd-base.msi) (once installed, `fleetd` and `fleet-desktop` will be upgraded to the latest)

2. Install fleet on Windows boxes by passing the `FLEET_URL` and `FLEET_SECRET` properties to the MSI installer:

```xml
msiexec /i fleetd-base.msi FLEET_URL="<target_url>" FLEET_SECRET="<secret_to_use>"
```

These steps are a flexible alternative to deploying Fleet's agent across macOS and Windows platforms. This method, focused on separating the configuration from the `fleetd` package, empowers you with more control and simplifies the management of your device enrollments.

This approach complements the original packaging method, allowing you to choose the best fit for your organization’s needs. Whether you prioritize streamlined package generation or prefer granular control over configuration distribution, these methods foster an open, flexible environment for deploying Fleet.

We encourage you to explore this alternative method in your environment and see how it aligns with your operational workflows. If you have any questions, insights, or experiences to share, feel free to join our community [Fleet Slack channels](https://fleetdm.com/support). Your feedback helps us improve and fosters a collaborative space where ideas and solutions can flourish.


<meta name="articleTitle" value="Config-less fleetd agent deployment">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-01-31">
<meta name="articleImageUrl" value="../website/assets/images/articles/config-less-fleetd-agent-deployment-1600x900@2x.png">
<meta name="description" value="Config-less `fleetd` agent deployment">
