# Fleet Desktop

Fleet Desktop is a self-service portal for your end users. It shows up in the menu bar on macOS and system tray on Windows/Linux. Learn more about [Linux support](https://fleetdm.com/docs/get-started/faq#what-host-operating-systems-does-fleet-support).

Fleet Desktop unlocks two key benefits:

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/cI2vDG3PbVo" allowfullscreen></iframe>
</div>

If your end users have a hard time finding Fleet Desktop in the macOS menu bar, you can optionally deploy the [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin). Additionally, to remind end users that they're failing policies, you can deploy [this configuration profile](https://github.com/fleetdm/fleet/blob/8cd2da576b01075db63d0a254ae597291c1d3d96/it-and-security/lib/macos/configuration-profiles/fleet-desktop-login-item.mobileconfig) to open this app everytime the end user logs in or restarts their Mac. 

## Install and upgrade

Fleet Desktop is bundled in Fleet's agent (fleetd) so it's automatically installed when you [enroll hosts](https://fleetdm.com/guides/enroll-hosts) by installing fleetd.

Once installed, Fleet Desktop will be [automatically upgraded](https://fleetdm.com/guides/fleetd-updates).

## Custom transparency link

Organizations with complex security postures can direct end users to a resource of their choice to serve custom content.

> The custom transparency link is only available for users with Fleet Premium

To turn on the custom transparency link in the Fleet UI, click on your profile in the top right and select **Settings**.
On the settings page, go to **Organization Settings > Fleet Desktop > Custom transparency URL**.

For information on setting the custom transparency link via a YAML configuration file, see the [configuration files](https://fleetdm.com/docs/configuration/yaml-files#fleet-desktop) documentation.

## Security

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

## Advanced

### Hide the menu bar icon on macOS

Some Fleet users want to hide the menu bar icon on macOS because of the limited menu bar "real estate."

How to hide the menu bar icon:

1. Add the [hide script](TODO) to Fleet.
2. Go to a device's **Host details** page.
3. Select **Actions > Run script** and run the hide script.

To run this script automatically across all macOS hosts create a policy in Fleet with the following query:

```sql
TODO
```

Then, add connect hide script to this policy via [policy automations](https://fleetdm.com/guides/policy-automation-run-script). 

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-04-19">
<meta name="articleTitle" value="Fleet Desktop">
<meta name="description" value="Learn about Fleet Desktop's features for self-remediation and transparency.">
