# Fleet Desktop

Fleet Desktop is a self-service portal for your end users. On macOS, it's an app. On Windows and [Linux](https://fleetdm.com/docs/get-started/faq#what-host-operating-systems-does-fleet-support), it shows up in the system tray. A Fleet Desktop app for Windows is [coming soon](https://github.com/fleetdm/fleet/issues/50162).

Fleet Desktop unlocks two key benefits:

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/cI2vDG3PbVo" allowfullscreen></iframe>
</div>

## Deploy Fleet Desktop

Fleet Desktop is deployed automatically with Fleet's agent (fleetd) when you [enroll hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#fleet-desktop). It shows up in the menu bar on macOS and the system tray on Windows and Linux.

### macOS

The best practice on macOS is to deploy the [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin) and [hide the menu bar icon](#hide-the-menu-bar-icon-on-macos). The app is required for [patch notifications](https://fleetdm.com/guides/patching-end-user-experience#force-patch-notify-before-patching) and Fleet's [Platform SSO extension](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet).

1. Add the Fleet Desktop app as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps#add-a-fleet-maintained-app).
2. Install it on new Macs during [setup experience](https://fleetdm.com/guides/setup-experience#install-software).
3. Install it on existing Macs with a policy. Create a policy with the query below, then add the Fleet Desktop app to the policy's [install software automation](https://fleetdm.com/guides/automatic-software-install-in-fleet). If an end user deletes the app, Fleet reinstalls it.

```sql
SELECT 1 FROM apps WHERE path = '/Applications/Fleet Desktop.app' AND bundle_identifier = 'com.fleetdm.fleet-desktop';
```

4. Optional: To open the app every time the end user logs in or restarts their Mac, deploy [this configuration profile](https://github.com/fleetdm/fleet/blob/8cd2da576b01075db63d0a254ae597291c1d3d96/it-and-security/lib/macos/configuration-profiles/fleet-desktop-login-item.mobileconfig).

> If an end user quits the app, it stays closed until they log in again.

## Update Fleet Desktop

fleetd updates Fleet Desktop automatically. To learn more, visit: [Self-managed agent updates](https://fleetdm.com/docs/deploying/fleetctl-agent-updates#self-managed-agent-updates).

### macOS

The Fleet Desktop app is a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps#update-apps-automatically), so fleetd doesn't update it. To keep it up to date, add a [patch policy](https://fleetdm.com/guides/patching-end-user-experience) for it.

## Custom transparency link
Organizations with complex security postures can direct end users to a resource of their choice to serve custom content.

> The custom transparency link is only available for users with Fleet Premium

To turn on the custom transparency link in the Fleet UI, click on your profile in the top right and select **Settings**.
On the settings page, go to **Organization Settings > Fleet Desktop > Custom transparency URL**.

For information on setting the custom transparency link via a YAML configuration file, see the [configuration files](https://fleetdm.com/docs/configuration/yaml-files#fleet-desktop) documentation.

## Secure Fleet Desktop

Requests sent by Fleet Desktop and the web page that opens when clicking on the "My Device" tray item use a [Random (Version 4) UUID](https://www.rfc-editor.org/rfc/rfc4122.html#section-4.4) token to uniquely identify each host.

The server uses this token to authenticate requests that give host information. Fleet uses rate limiting and token rotation to secure access to this information.

Successfully brute-forcing this UUID is about [as likely as you getting hit by a meteorite this year](https://pkg.go.dev/github.com/google/uuid#NewRandom).

**Rate limiting**

To prevent brute-forcing attempts, Fleet rate-limits the endpoints used by Fleet Desktop on a per-IP basis. If an IP requests more than 1000 **consecutive** invalid UUIDs in a one-minute interval, Fleet will ban requests from such IP for one minute (fail requests with HTTP error code 429). This rate limit algorithm is used to support deployments of Fleet where all hosts are behind the same NAT (all hosts mapped to the same IP).

**Token rotation**

```
ℹ️  In Fleet v4.22.0, token rotation for Fleet Desktop was introduced.
```

Starting with Fleet v4.22.0, the server will reject any token older than one hour since it was issued. This helps Fleet protect against unintentionally leaked or brute-forced tokens.

As a consequence, Fleet Desktop will issue a new token if the current token is:

- Rejected by the server
- Older than one hour

This change is imperceptible to users, as clicking on the "My device" tray item always uses a valid token. If a user visits an address with an expired token, they will get a message instructing them to click on the tray item again.

**Single Sign-On (SSO)**

Fleet can be configured to require an end user to authenticate via SSO to view the "My Device" page. 

To require SSO to view the "My Device" page, click on your profile in the top right and select **Settings**.
On the settings page, go to **Organization Settings > Fleet Desktop > End user authentication**.

> If you deploy the [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin), end users have to sign in each time the app is relaunched (for example, after an app update, a restart, or quitting the app). The app doesn't share a session with the Fleet Desktop menu bar icon or the end user's web browser, so signing in to one doesn't sign them in to the others.

## Advanced

### Hide the menu bar icon on macOS

If you deploy the [Fleet Desktop app](#deploy-the-fleet-desktop-app-on-macos), hide the menu bar icon so end users have one place to find self-service.

Hiding the icon stops the Fleet Desktop menu bar process entirely, so Fleet doesn't show a Fleet Desktop version for these hosts. The Fleet Desktop app and patch notifications keep working.

If you're enrolling Macs manually, generate a Fleet agent (fleetd) without the Fleet Desktop menu bar icon by excluding the `--fleet-desktop` flag from the `fleetctl package` command.

If you're automatically enrolling Macs via Apple Business, hide the icon with a script:

1. Add this script to Fleet:

```sh
#!/bin/sh
# Hide the Fleet Desktop menu bar icon on Macs that have Fleet Desktop.app.
# Sets ORBIT_FLEET_DESKTOP to false, then reloads orbit from a temporary LaunchDaemon,
# because orbit runs this script and booting it out directly would kill the script too.

ORBIT_PLIST="/Library/LaunchDaemons/com.fleetdm.orbit.plist"
RELOAD_LABEL="com.fleetdm.hide-fleet-desktop-menu-bar"

if [ ! -f "$ORBIT_PLIST" ]; then
    echo "Fleet's agent (fleetd) is not installed."
    exit 1
fi

/usr/bin/plutil -replace EnvironmentVariables.ORBIT_FLEET_DESKTOP -string "false" "$ORBIT_PLIST" || exit 1

dir=$(/usr/bin/mktemp -d /private/tmp/fleet-hide-desktop.XXXXXX) || exit 1

cat << EOF > "$dir/reload.sh"
#!/bin/sh
/bin/sleep 15
/bin/launchctl bootout system "$ORBIT_PLIST"
/usr/bin/pkill -x fleet-desktop
count=0
until /bin/launchctl bootstrap system "$ORBIT_PLIST"; do
    count=\$((count + 1))
    [ "\$count" -ge 30 ] && break
    /bin/sleep 1
done
# If orbit still isn't loaded, exit nonzero so launchd reruns this job (KeepAlive) until it is.
/bin/launchctl print "system/com.fleetdm.orbit" > /dev/null 2>&1 || exit 1
/bin/launchctl kickstart "system/com.fleetdm.orbit"
/bin/rm -rf "$dir"
/bin/launchctl bootout "system/$RELOAD_LABEL"
EOF

cat << EOF > "$dir/reload.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$RELOAD_LABEL</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>$dir/reload.sh</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
</dict>
</plist>
EOF

if ! /bin/launchctl bootstrap system "$dir/reload.plist"; then
    /bin/rm -rf "$dir"
    echo "Couldn't schedule the fleetd reload. Is one already pending?"
    exit 1
fi

echo "Fleet Desktop menu bar icon will be hidden in about 15 seconds."
```

2. Create a policy with the following query. It passes when the menu bar icon is hidden and fleetd has reloaded:

```sql
SELECT 1 FROM plist p, orbit_info o
WHERE p.path = '/Library/LaunchDaemons/com.fleetdm.orbit.plist'
  AND p.key = 'EnvironmentVariables'
  AND p.subkey = 'ORBIT_FLEET_DESKTOP'
  AND p.value = 'false'
  AND o.desktop_version = '';
```

3. Target the policy to a label that includes only Macs with the Fleet Desktop app installed, so Macs without the app keep the menu bar icon. For example, a label with this query:

```sql
SELECT 1 FROM apps WHERE path = '/Applications/Fleet Desktop.app';
```

4. Add the hide script to the policy via [policy automations](https://fleetdm.com/guides/policy-automation-run-script).

Fleet's agent (fleetd) upgrades won't re-show the menu bar icon, because upgrades don't touch the plist the script updates.


### Single sign-on (SSO) for Fleet-Desktop-token-authenticated routes

_Available in Fleet Premium_

Fleet Desktop's API routes authenticate with a per-device token, not your Fleet API token. Turning on [`fleet_desktop.sso_enabled`](https://fleetdm.com/docs/configuration/yaml-files#fleet-desktop) adds a second requirement: signing in through your IdP.

See the [Fleet-desktop-token-authenticated routes reference](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#fleet-desktop-token-authenticated-routes) for how the session requirement, error responses, and exemptions (including during [setup experience](https://fleetdm.com/guides/setup-experience)) work.


### How iOS/iPadOS hosts authenticate

iOS and iPadOS hosts don't run a Fleet Desktop app. Instead, end users reach a self-service page through a [Web Clip configuration profile](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-and-ipados). Fleet authenticates the request via host UUID in the URL.

This UUID-based authentication only works for iOS and iPadOS hosts and is supported for all [device-authenticated API routes](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#device-authenticated-routes) except the following (neither of which applies to iOS/iPadOS hosts):
- [Reporting an agent error](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#report-an-agent-error)
- [Download device's MDM manual enrollment profile](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#download-devices-mdm-manual-enrollment-profile)

Because iOS and iPadOS hosts authenticate by UUID, Fleet also strips identifying details, like hardware serial and configuration profile data, from the [Get host by Fleet Desktop token](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-fleet-desktop-token) response for those hosts.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-04-19">
<meta name="articleTitle" value="Fleet Desktop">
<meta name="description" value="Learn about Fleet Desktop's features for self-remediation and transparency.">
