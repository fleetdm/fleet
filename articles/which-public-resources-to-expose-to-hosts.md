# Which public resources to expose to hosts?

Some organizations block all outbound internet traffic by default and only let hosts reach the internet through a VPN or other secure, managed network. If that's your setup, you need to explicitly allow a small set of public resources so hosts can enroll, stay managed by Fleet, receive OS updates, and keep Fleet's agent (fleetd) up to date.

This guide lists those resources. Add them as exceptions in your VPN, proxy, or firewall's allowlist.

## Fleet

- Your Fleet server: Fleet's agent (fleetd) checks in with your Fleet server to run queries and policies, install software, and receive MDM commands. See [Which API endpoints to expose to the public internet?](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet) for the exact paths to allow.
- `download.fleetdm.com`: Hosts the public fleetd base installers (`.pkg`, `.msi`, `.deb`, and `.rpm`) used to enroll new hosts.
- `updates.fleetdm.com`: Fleet's [The Update Framework (TUF)](https://theupdateframework.io/) server. Fleetd uses this for auto-updates.
  - If you'd rather not expose this host, run [your own TUF update server](https://fleetdm.com/guides/fleetd-updates) with a Fleet Premium license.

## Apple

If you manage macOS, iOS, or iPadOS hosts, those hosts need direct access to Apple's own services, separate from Fleet's. This is especially true for hosts enrolled with [Automated Device Enrollment (ADE)](https://support.apple.com/guide/deployment/automated-device-enrollment-management-dep73069dd57/web).

Apple maintains the [definitive, current list](https://support.apple.com/en-us/101555). At minimum, allow:

- `*.push.apple.com`: Apple Push Notification service (APNs). Fleet uses this to deliver MDM commands to hosts.
- `deviceenrollment.apple.com`, `mdmenrollment.apple.com`, and `iprofiles.apple.com`: Deliver enrollment profiles during Automated Device Enrollment.
- `gdmf.apple.com` and `identity.apple.com`: Device management catalog lookups and APNs certificate requests.
- `vpp.itunes.apple.com`: Assigning and revoking Apps and Books licenses.
- The hosts listed under "Device setup" and "Software updates" in [Apple's list](https://support.apple.com/en-us/101555), if you use Fleet to enforce OS updates.

## Microsoft

If you manage Windows hosts, especially ones enrolled with Windows Autopilot, see Microsoft's [Windows Autopilot requirements](https://learn.microsoft.com/en-us/intune/autopilot/networking-requirements) for the hosts Windows needs to reach directly.

## Google

If you manage Android hosts, see Google's [Android Enterprise network requirements](https://support.google.com/android/work/answer/10513641) for the hosts Android needs to reach directly. You'll also need `/api/fleetd/*` exposed on your Fleet server if you [connect end users to Wi-Fi or VPN with a certificate](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2026-07-01">
<meta name="articleTitle" value="Which public resources to expose to hosts?">
<meta name="description" value="Which Fleet, Apple, Microsoft, and Google resources to allow when hosts can only reach the internet through a VPN or secure network.">
