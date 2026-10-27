# Autopilot without Autopilot: zero-touch Windows deployment with Fleet

*Windows Autopilot ships laptops straight to end users, but it expects you to pay for Microsoft Entra ID P1 for every one of them. A provisioning package and Fleet get new devices enrolled from the first boot without that bill.*

## Key takeaways

- **Zero touch doesn't have to mean Autopilot.** A Windows provisioning package installs Fleet's agent at the first setup screen, so a new laptop enrolls in Fleet the first time it gets online, before the end user reaches the desktop.
- **You skip the per-user Entra ID P1 license.** Autopilot and Entra automatic enrollment both need a P1 license for each end user. At Microsoft's list price, that's $84 per person per year that this approach doesn't require.
- **There's no device registration step.** Any Windows Pro, Enterprise, or Education device straight out of the box can take the package. You don't need your reseller to register hardware with Microsoft first.
- **A partner can make it true zero touch.** Resellers and OEMs that offer staging services can apply the package in their warehouse, so devices go straight to end users without passing through IT.
- **Or IT can do it in a few minutes per device.** Apply the package from a USB drive, confirm the host shows up in Fleet, power the device off, and ship it.
- **The trade-offs are real, and manageable.** Devices aren't joined to Entra ID, Windows MDM turns on at the first sign-in, and the package carries your enroll secret, so plan for each.

<a purpose="cta-button" href="https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package">Read the step-by-step guide</a>

Windows Autopilot is Microsoft's answer to a question every IT team asks: how do you ship a laptop straight to a new hire and still have it managed from the first boot? The end user signs in with their work account, Windows joins Microsoft Entra ID, and the device enrolls in your MDM before they reach the desktop.

It works well when you already pay for the Microsoft identity stack underneath it. If you don't, or you'd rather not pay per person to get devices enrolled, Windows has another path built in.

## How zero touch works without Autopilot

Windows has supported provisioning packages (`.ppkg` files) since Windows 10. A package is a bundle of settings and installers that you build with Microsoft's Windows Configuration Designer app. Insert a USB drive with a package at the first Windows setup screen, and Windows applies it before anyone creates an account.

For Fleet, the package carries one thing: Fleet's agent (fleetd), built for the fleet you want new devices to join. Windows installs fleetd during setup, fleetd enrolls the device in Fleet as soon as it's online, and the device powers back off while it's still sitting at the first setup screen. When the end user unboxes it and finishes Windows setup, the device is already in Fleet, and anything you've set to install or configure automatically starts right away.

You don't give up MDM, either. With [Windows MDM turned on](https://fleetdm.com/guides/windows-mdm-setup), Fleet turns on MDM for hosts that enroll through fleetd, so configuration profiles, disk encryption, and OS updates still work without Entra ID.

## What you stop paying for

### Entra ID P1 for every end user

To connect Fleet to Entra ID for automatic enrollment or Autopilot, [Microsoft requires](https://fleetdm.com/guides/windows-mdm-setup#microsoft-licenses-you-need) a qualifying subscription for your tenant plus a Microsoft Entra ID P1 license for each end user who enrolls a device. Microsoft [lists Entra ID P1](https://www.microsoft.com/en-us/security/business/microsoft-entra-pricing) at $7.00 per user per month on an annual commitment.

That adds up. For a 500-person company, it's $42,000 a year, spent so that devices can enroll. A provisioning package doesn't depend on Entra ID at all, so that line item goes away.

To be fair to Autopilot: if your end users already have Microsoft 365 E3 or E5, Entra ID P1 is already included, and the license savings don't apply to you. The rest of this article still might.

### Registering devices in advance

Autopilot only works on devices that are [registered with Windows Autopilot](https://learn.microsoft.com/en-us/autopilot/requirements?tabs=configuration) before they ship, either by your OEM or reseller or by uploading each device's hardware hash. That's one more dependency on your supply chain, and one more place for a device to fall through the cracks.

A provisioning package has nothing to register. It works on any Windows Pro, Enterprise, or Education device that's still at the first setup screen, including one you bought at a retail store yesterday.

## Make it true zero touch with a partner

Someone has to insert the USB drive at the first setup screen. If that someone works for your reseller or OEM, IT never touches the device.

Many hardware resellers and OEMs offer staging or configuration services: imaging, asset tagging, and applying your settings before devices leave their warehouse. Ask whether theirs can apply a Windows provisioning package at the first setup screen and power the device off afterward, without completing setup. Then hand over the package and its password, and devices ship straight from the warehouse to your end users.

The partner doesn't need access to Fleet. If they can't confirm enrollment before shipping, the device checks in, completes enrollment, and applies anything you've set to configure automatically as soon as the end user connects it to Wi-Fi.

Because the package contains your enroll secret, consider building the partner's package with its own enroll secret. Fleet supports more than one enroll secret per fleet, so you can rotate the partner's secret without affecting anything else.

## Or do it yourself before you ship

You don't need a partner to get most of the benefit. If devices already pass through IT before they go out, applying the package takes a few minutes per device: insert the USB drive, enter the package password, wait for fleetd to install and enroll, then power the device off and ship it.

Doing it in-house has one advantage: you can confirm each host has enrolled in Fleet before it leaves the building. The [step-by-step guide](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package) walks through building fleetd, creating the package in Windows Configuration Designer, and applying it.

## Know the trade-offs

This isn't a drop-in replacement for everything Autopilot does, and it's worth being clear about the differences before you commit:

- **Devices aren't joined to Entra ID.** If you need Entra-joined devices, or end users signing in to Windows with their Entra work account from day one, Autopilot or Entra automatic enrollment is still the right tool.
- **Windows MDM turns on after the first sign-in.** Fleet enrolls the device right away, but Windows [finishes MDM enrollment](https://fleetdm.com/guides/windows-mdm-setup#manual-enrollment) once someone signs in. Until then, configuration profiles stay queued.
- **There's no identity check at enrollment.** Windows setup has no web browser, so the package can't ask end users to sign in to your identity provider when the device enrolls.
- **The package carries your enroll secret.** Encrypt the package, control who has it, and rotate the secret if it leaves your hands.
- **Someone still touches each device once.** Either a partner or IT applies the package, so plan for that step in your procurement flow.
- **Windows Home isn't supported for MDM.** Buy Pro, Enterprise, or Education editions.

## Zero touch is a workflow, not a license

Autopilot is a good fit if you already pay for Entra ID P1 and want Entra-joined devices. For everyone else, a provisioning package, Fleet, and either a partner's warehouse or a few minutes of IT time get you to the same outcome: laptops that ship straight to end users and show up in Fleet the first time they're online.

## See it live

- Follow the guide: [Preinstall Fleet's agent on Windows with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package)
- **Get a demo:** [talk to Fleet](https://fleetdm.com/contact) about zero-touch Windows deployment.

<meta name="articleTitle" value="Autopilot without Autopilot: zero-touch Windows deployment with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-09-25">
<meta name="description" value="Get Autopilot-style zero-touch Windows deployment without per-user Entra ID P1 licenses, using a provisioning package and Fleet.">
