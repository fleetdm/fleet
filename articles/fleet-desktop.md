# Fleet Desktop

Fleet Desktop is a menu bar icon available on macOS, Windows, and Linux that gives your end users visibility into the security posture of their machine. This unlocks two key benefits:

* Self-remediation: end users can see which policies they are failing and resolution steps, reducing the need for IT and security teams to intervene
* Scope transparency: end users can see what the Fleet agent can do on their machines, eliminating ambiguity between end users and their IT and security teams

> Self-remediation is only available for users with Fleet Premium

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/cI2vDG3PbVo" allowfullscreen></iframe>
</div>

## Install Fleet Desktop
For information on how to install Fleet Desktop, visit: [Adding Hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#fleet-desktop).

## Upgrade Fleet Desktop
Once installed, Fleet Desktop will be automatically updated via Fleetd. To learn more, visit: [Self-managed agent updates](https://fleetdm.com/docs/deploying/fleetctl-agent-updates#self-managed-agent-updates).

## Custom transparency link
For organizations with complex security postures, they can direct end users to a resource of their choice to serve custom content.

> The custom transparency link is only available for users with Fleet Premium

To turn on the custom transparency link in the Fleet GUI, click on your profile in the top right and select "Settings."
On the settings page, go to "Organization Settings" and select "Fleet Desktop." Use the "Custom transparency URL" text input to specify the custom URL.

For information on how to set the custom transparency link via a YAML configuration file, see the [configuration files](https://fleetdm.com/docs/configuration/fleet-server-configuration#fleet-desktop-settings) documentation.

## Secure Fleet Desktop

Requests sent by Fleet Desktop and the web page that opens when clicking on the "My Device" tray item use a [Random (Version 4) UUID](https://www.rfc-editor.org/rfc/rfc4122.html#section-4.4) token to uniquely identify each host.

The server uses this token to authenticate requests that give host information. Fleet uses the following methods to secure access to this information.

**Rate limiting**

To prevent brute-forcing, Fleet rate-limits the endpoints used by Fleet Desktop on a per-IP basis. If an IP requests more than 720 invalid UUIDs in a one-hour interval, Fleet will return HTTP error code 429.

**Token rotation**

```
ℹ️  In Fleet v4.22.0, token rotation for Fleet Desktop was introduced.
```

Starting with Fleet v4.22.0, the server will reject any token older than one hour since it was issued. This helps Fleet protect against unintentionally leaked or brute-forced tokens.

As a consequence, Fleet Desktop will issue a new token if the current token is:

- Rejected by the server
- Older than one hour

This change is imperceptible to users, as clicking on the "My device" tray item always uses a valid token. If a user visits an address with an expired token, they will get a message instructing them to click on the tray item again.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-04-19">
<meta name="articleTitle" value="Fleet Desktop">
<meta name="description" value="Learn about Fleet Desktop's features for self-remediation and transparency.">
