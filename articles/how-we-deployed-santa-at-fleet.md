# How we deployed Santa at Fleet

### Links to article series:

- Part 1: [Deploy Santa with Fleet GitOps and skip the sync server](https://fleetdm.com/articles/deploy-santa-with-fleet-gitops-and-skip-the-sync-server)
- Part 2: How we deployed Santa at Fleet

Here's a real-world example of deploying Santa with the Fleet GitOps approach that we are using internally:

**Step 1: Deploy the Santa app via Fleet GitOps**

Santa is a [Fleet-maintained app](https://fmalibrary.com/) making deployment easy. Alternatively, you can use our deployment YAML file: https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/software/santa.yml

Either method allows the Santa app to be installed on a test device group through self service. This can be changed to fit your needs, for example, by using automatic deployment instead of self service or deploying to multiple labels. 

**Step 2: Deploy the Santa configuration**

Our suggested best practice is to deploy two Configuration Profiles: one for managing the Santa app configuration and the other for managing Santa rules. Keeping the two configurations modular and separate minimizes the risk of Santa rules changes from interfering with the app config.

Here's a snippet from our [Santa configuration profile](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/santa-configuration.mobileconfig):

```xml
<dict>
    <key>BannedBlockMessage</key>
    <string>This application has been blocked by a security policy.</string>
    <key>ClientMode</key>
    <integer>1</integer>
    <key>FileChangesRegex</key>
    <string>^/(?!(?:private/tmp|Library/(?:Caches|Managed Installs/Logs|(?:Managed )?Preferences))/)</string>
    <key>MachineIDKey</key>
    <string>MachineUUID</string>
    <key>MachineIDPlist</key>
    <string>/Library/Preferences/com.company.machine-mapping.plist</string>
    <key>MachineOwnerKey</key>
    <string>Owner</string>
    <key>MachineOwnerPlist</key>
    <string>/Library/Preferences/com.company.machine-mapping.plist</string>
    <key>ModeNotificationLockdown</key>
    <string>Entering Lockdown mode</string>
    <key>ModeNotificationMonitor</key>
    <string>Entering Monitor mode&lt;br/&gt;Please be careful!</string>
    <key>SyncBaseURL</key>
    <string></string>
</dict>
```

Here's a snippet from our [Santa rules configuration profile](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/santa-rules.mobileconfig):

```xml
<key>StaticRules</key>
<array>
    <dict>
        <!-- Always allow files signed by North Pole Security Inc -->
        <key>identifier</key>
        <string>ZMCG7MLDV9</string>
        <key>policy</key>
        <string>ALLOWLIST</string>
        <key>rule_type</key>
        <string>TEAMID</string>
    </dict>
    <dict>
        <!-- Always BLOCK the BundleExample.app binary in Santa's testdata files, for testing -->
        <key>identifier</key>
        <string>b7c1e3fd640c5f211c89b02c2c6122f78ce322aa5c56eb0bb54bc422a8f8b670</string>
        <key>policy</key>
        <string>BLOCKLIST</string>
        <key>rule_type</key>
        <string>BINARY</string>
    </dict>
    <dict>
        <!-- Block WhatsApp.app -->
        <key>identifier</key>
        <string>54a8ec11bcea48a276b1fdce556a29108ba77de4</string>
        <key>policy</key>
        <string>BLOCKLIST</string>
        <key>rule_type</key>
        <string>CDHASH</string>
    </dict>
</array>
```

**Step 3. Deploy Santa Extensions**

Policy to check if extensions exist: https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/policies/install-santa-extension.yml 

Script to install the extensions: https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/install-santa-extension.sh

We chose to deploy these via policy automation since the Santa extensions don't exist natively in Fleet. We have a policy that checks for the existence of the Santa extension. If it is not found, Fleet will immediately run a remediation script that handles the download and configuration of fleetd to start using the extension.

**Step 4. Collect Santa Events**

Collect Santa denied logs: https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/queries/collect-santa-denied-logs.yml

Here we are collecting every time a device attempts to open a blocklisted app. These logs can then be collected in our SIEM or used to fire a webhook to alert us via Slack.

![siem-dashboard](../website/assets/images/articles/siem-dashboard-700x495@2x.png)

_SIEM dashboard_

![slack-message](../website/assets/images/articles/slack-message-700x407@2x.png)

_Slack message_

## The GitOps advantage

![pull-request](../website/assets/images/articles/pull-request-1000x425@2x.png)

_Pull Request adding an additional rule_

By leveraging GitOps principles through Fleet, Santa management becomes:

- **Version Controlled:** Every rule change is tracked in Git with full audit trails
- **Peer Reviewed:** Pull request workflows ensure rule changes are reviewed before deployment
- **Automatically Deployed:** CI/CD pipelines handle rule distribution without manual intervention
- **Easily Rollbacked:** Git reverts enable instant rollback of problematic rule changes

About the author: [Allen Houchins](https://www.linkedin.com/in/allenhouchins/) is a Solutions Consultant / individual contributor and head of IT at Fleet Device Management.

<meta name="articleTitle" value="How we deployed Santa at Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-02-24">
<meta name="description" value="Part 2 of 2 - Learn to manage Santa in a whole new way with less complexity and overhead.">
