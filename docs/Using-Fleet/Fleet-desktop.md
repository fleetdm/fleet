# Fleet Desktop
- [Installing Fleet Desktop](#installing-fleet-desktop)
- [Upgrading Fleet Desktop](#upgrading-fleet-desktop)
- [Custom Transparency Link](#custom-transparency-link)
- [Securing Fleet Desktop](#securing-fleet-desktop)

Fleet Desktop is a menu bar icon available on macOS, Windows, and Linux.

At its core, Fleet Desktop gives your end users visibility into the security posture of their machine. This unlocks two key benefits:
* Self-remediation: end users can see which policies they are failing and resolution steps, reducing the need for IT and security teams to intervene
* Scope Transparency: end users can see what the Fleet agent can do on their machines, eliminating ambiguity between end users and their IT and security teams

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/cI2vDG3PbVo" allowfullscreen></iframe>
</div>

## Installing Fleet Desktop
For information on how to install Fleet Desktop, visit: [Adding Hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#fleet-desktop).

## Upgrading Fleet Desktop
Once installed, Fleet Desktop will be automatically updated via Orbit. To learn more, visit: [Self-managed agent updates](https://fleetdm.com/docs/deploying/fleetctl-agent-updates#self-managed-agent-updates).

## Custom transparency link
For organizations with complex security postures, they can direct end users to a resource of their choice to serve custom content.

> The custom transparency link is only available for users with Fleet Premium

To turn on the custom transparency link in the Fleet GUI, click on your profile in the top right and select "Settings."
On the settings page, go to "Organization Settings" and select "Fleet Desktop." Use the "Custom transparency URL" text input to specify the custom URL.

For information on how to set the custom transparency link via a YAML configuration file, see the [configuration files](../Using-Fleet/configuration-files/README.md#fleet-desktop-settings) documentation.

## Securing Fleet Desktop

To prevent brute-forcing, Fleet rate-limits the endpoints used by Fleet Desktop on a per-IP basis. If an IP requests more than 720 invalid UUIDs in a one-hour interval, Fleet will return HTTP error code 429.

<meta name="title" value="Fleet Desktop">
<meta name="pageOrderInSection" value="450">
